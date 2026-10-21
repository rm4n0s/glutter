//go:build js && wasm

package webgpu

// helperScript is the small piece of JavaScript this package keeps: promises cannot be attached
// from Go without a js.Func for each one, and a process can have only so many of them, so the
// script attaches them and calls the callbacks that Go made once, with the number of the job,
// the device or the read (the way internal/platform/web does for the clipboard). It returns the
// object of its functions.
//
//   - request(codes, job): asks for an adapter, a device and the compiled WGSL modules, and
//     calls acquired(job, result) or failed(job, tag, message); never waits for anyone
//   - watch(device, id): calls lost(id, reason, message) when the device is lost and
//     uncaptured(id, message) for every error that no scope caught
//   - map(buffer, id): calls mapped(id) or mapFailed(id, message) when the buffer is mapped
//   - preferredFormat(): the format of the canvas
const helperScript = `
var cb = null;
var h = {};
h.init = function (callbacks) { cb = callbacks; };
h.preferredFormat = function () { return navigator.gpu.getPreferredCanvasFormat(); };
h.request = function (codes, job) {
	(async function () {
		var gpu = (typeof navigator !== 'undefined') ? navigator.gpu : undefined;
		if (!gpu) { throw {tag: 'unavailable', msg: 'navigator.gpu is not defined: this browser or this page has no WebGPU'}; }
		var adapter;
		try { adapter = await gpu.requestAdapter(); } catch (e) { throw {tag: 'unavailable', msg: 'requestAdapter() failed: ' + e}; }
		if (!adapter) { throw {tag: 'unavailable', msg: 'requestAdapter() answered null: the browser has no WebGPU adapter'}; }
		var device;
		try { device = await adapter.requestDevice(); } catch (e) { throw {tag: 'device', msg: 'requestDevice() failed: ' + e}; }
		var modules = {}, problems = [];
		for (var name of Object.keys(codes)) {
			var m = device.createShaderModule({label: name, code: codes[name]});
			var info = await m.getCompilationInfo();
			for (var msg of info.messages) {
				if (msg.type === 'error') { problems.push(name + ':' + msg.lineNum + ':' + msg.linePos + ': ' + msg.message); }
			}
			modules[name] = m;
		}
		if (problems.length > 0) {
			try { device.destroy(); } catch (e) {}
			throw {tag: 'shader', msg: problems.join('\n')};
		}
		var ai = adapter.info || {};
		var description = [ai.vendor, ai.architecture, ai.device, ai.description].filter(Boolean).join(' ');
		return {adapter: adapter, device: device, modules: modules, description: description, fallback: !!adapter.isFallbackAdapter};
	})().then(
		function (r) { cb.acquired(job, r); },
		function (e) { cb.failed(job, (e && e.tag) ? e.tag : 'device', (e && e.msg) ? e.msg : String(e)); }
	);
};
h.watch = function (device, id) {
	device.lost.then(function (info) { cb.lost(id, String(info.reason || ''), String(info.message || '')); });
	device.addEventListener('uncapturederror', function (ev) {
		var e = ev.error;
		cb.uncaptured(id, e ? ((e.constructor && e.constructor.name) || 'GPUError') + ': ' + e.message : 'unknown error');
	});
};
h.map = function (buffer, id) {
	buffer.mapAsync(1).then(function () { cb.mapped(id); }, function (e) { cb.mapFailed(id, String(e)); });
};
return h;
`
