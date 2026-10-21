// The loader of a Glutter web app (SPEC §23.1). It copies the query parameters of the
// page's URL whose names start with GLUTTER_ into the Go program's environment, so that
// ?GLUTTER_RENDERER=webgl works as the environment variable does on Linux, then fetches
// app.wasm with WebAssembly.instantiateStreaming and runs it.
//
// glutterEnvFromQuery follows platform/web.EnvFromQuery (Go); loader_test.go runs both on
// the same inputs.

function glutterEnvFromQuery(search) {
  const env = {};
  const query = search.charAt(0) === "?" ? search.slice(1) : search;
  if (query === "") return env;
  for (const pair of query.split("&")) {
    const eq = pair.indexOf("=");
    const rawName = eq < 0 ? pair : pair.slice(0, eq);
    const rawValue = eq < 0 ? "" : pair.slice(eq + 1);
    let name, value;
    try {
      name = decodeURIComponent(rawName.replace(/\+/g, " "));
      value = decodeURIComponent(rawValue.replace(/\+/g, " "));
    } catch (e) {
      continue; // a bad escape: leave the parameter out
    }
    if (!name.startsWith("GLUTTER_") || name.indexOf("=") >= 0 || name.indexOf("\0") >= 0 || value.indexOf("\0") >= 0) continue;
    env[name] = value;
  }
  return env;
}

// glutterShow puts a message into the page, for a browser that cannot run the app.
function glutterShow(text) {
  const p = document.createElement("p");
  p.id = "glutter-message";
  p.style.cssText = "font:16px system-ui,sans-serif;margin:2em;color:#333";
  p.textContent = text;
  document.body.appendChild(p);
}

// glutterReportURL is where to post how the program ended, when the page was opened with
// the parameter glutter_report (a path on the server); the browser tests use it.
function glutterReportURL(search) {
  const m = /(?:^|[?&])glutter_report=([^&]*)/.exec(search);
  return m ? decodeURIComponent(m[1]) : "";
}

async function glutterBoot() {
  if (typeof WebAssembly !== "object" || typeof WebAssembly.instantiate !== "function") {
    glutterShow("This browser does not support WebAssembly, which this app needs.");
    return;
  }
  const report = glutterReportURL(location.search);
  const go = new Go();
  go.env = Object.assign({}, go.env, glutterEnvFromQuery(location.search));
  let log = "";
  if (report && globalThis.fs && globalThis.fs.writeSync) {
    // Keep what the program writes to standard output and error, to report it when it ends.
    const write = globalThis.fs.writeSync.bind(globalThis.fs);
    const decoder = new TextDecoder("utf-8");
    globalThis.fs.writeSync = (fd, buf) => {
      if (log.length < 65536) log += decoder.decode(buf);
      return write(fd, buf);
    };
  }
  const finish = (code, message) => {
    if (!report) return;
    fetch(report, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ kind: "exit", exitCode: code, message: message || "", log }),
      keepalive: true,
    }).catch(() => {});
  };
  const exit = go.exit;
  go.exit = (code) => {
    finish(code, "");
    exit(code);
  };
  try {
    let result;
    if (WebAssembly.instantiateStreaming) {
      result = await WebAssembly.instantiateStreaming(fetch("app.wasm"), go.importObject);
    } else {
      const response = await fetch("app.wasm");
      result = await WebAssembly.instantiate(await response.arrayBuffer(), go.importObject);
    }
    await go.run(result.instance);
  } catch (e) {
    glutterShow("The app could not start: " + e);
    finish(-1, String(e));
  }
}

if (typeof document !== "undefined" && typeof Go === "function") {
  glutterBoot();
}
