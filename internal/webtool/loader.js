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

// glutterShow puts a message into the page: for a browser that cannot run the app, and for a
// program that stopped with an error (a start-up failure such as WebGPUUnavailable for a renderer
// that was asked for by name), which the user would otherwise find only in the console. It
// replaces an earlier message, and sits over the page so that a canvas that was already made
// does not hide it.
function glutterShow(text) {
  let p = document.getElementById("glutter-message");
  if (!p) {
    p = document.createElement("pre");
    p.id = "glutter-message";
    p.setAttribute("role", "alert");
    p.style.cssText = "position:fixed;left:0;right:0;top:0;z-index:2147483647;margin:0;padding:1em 2em;" +
      "font:14px/1.4 system-ui,sans-serif;white-space:pre-wrap;word-break:break-word;color:#5c0011;" +
      "background:#fff1f0;border-bottom:2px solid #b3261e;max-height:50vh;overflow:auto";
    (document.body || document.documentElement).appendChild(p);
  }
  p.textContent = text;
}

// glutterFailureText is the message for a program that ended with a non-zero exit code: the last
// lines of what it wrote to standard error that are not Info, Debug or Warn log lines (a failed
// RunApp is logged as an error line that names the tag, then as the last line), or the last lines
// when there are none. A line is cut at 300 characters: an error line can carry a stack.
function glutterFailureText(code, log) {
  const lines = String(log || "").split("\n").map((l) => l.trim()).filter((l) => l !== "");
  const important = lines.filter((l) => !/ (INFO|DEBUG|WARN) /.test(l));
  const tail = (important.length ? important : lines).slice(-4)
    .map((l) => (l.length > 300 ? l.slice(0, 300) + "..." : l)).join("\n");
  return "The app stopped with an error (exit code " + code + ")." + (tail ? "\n\n" + tail : "");
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
  if (globalThis.fs && globalThis.fs.writeSync) {
    // Keep what the program writes to standard output and error (the last 64 KiB), to show why it
    // failed and, when the page was opened with glutter_report, to report it.
    const write = globalThis.fs.writeSync.bind(globalThis.fs);
    const decoder = new TextDecoder("utf-8");
    globalThis.fs.writeSync = (fd, buf) => {
      log += decoder.decode(buf, { stream: true });
      if (log.length > 65536) log = log.slice(-65536);
      return write(fd, buf);
    };
  }
  const finish = (code, message) => {
    if (!report) return;
    const shown = document.getElementById("glutter-message");
    fetch(report, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ kind: "exit", exitCode: code, message: message || "", log, shown: shown ? shown.textContent : "" }),
      keepalive: true,
    }).catch(() => {});
  };
  const exit = go.exit;
  go.exit = (code) => {
    if (code !== 0) glutterShow(glutterFailureText(code, log));
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
