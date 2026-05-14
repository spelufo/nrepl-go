import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";
import { Type } from "typebox";
import { Text } from "@mariozechner/pi-tui";


export default function (pi: ExtensionAPI) {
  pi.registerTool({
    name: "nrepl-send",
    label: "nREPL",
    description: `Run the nrepl-send CLI. Arguments are forwarded directly to the binary.

Usage:
  nrepl-send eval <code> [-n <namespace>]   Evaluate Clojure code
  nrepl-send describe                       Show server info (versions, supported ops)
  nrepl-send completions <prefix>           Get symbol completions

Connection flags (optional, auto-discovered from .nrepl-port by default):
  --host <host>         nREPL host (default: localhost)
  --port <port>         nREPL port
  --port-file <path>    Path to .nrepl-port file`,
    promptSnippet: "Run the nrepl CLI to evaluate Clojure code and interact with a running nREPL server",
    parameters: Type.Object({
      args: Type.Array(Type.String(), {
        description: 'Arguments to pass to the nrepl binary, e.g. ["eval", "(+ 1 2)"] or ["eval", "(println :hi)", "-n", "my.ns"]',
      }),
    }),
    renderCall(args, theme, context) {
      const t = (context.lastComponent as Text | undefined) ?? new Text("", 0, 0);
      const argList: string[] = args.args ?? [];
      const cmd = argList[0] ?? "";
      let content = theme.fg("toolTitle", theme.bold("nREPL "));
      if (cmd === "eval") {
        const code = argList[1] ?? "";
        const nsIdx = argList.indexOf("-n");
        const ns = nsIdx >= 0 ? argList[nsIdx + 1] : null;
        content += theme.fg("accent", "eval");
        if (ns) content += theme.fg("dim", ` [${ns}]`);
        content += "\n" + theme.fg("dim", code);
      } else if (cmd === "completions") {
        content += theme.fg("accent", "completions ");
        content += theme.fg("dim", argList[1] ?? "");
      } else {
        content += theme.fg("accent", argList.join(" ") || "(no args)");
      }
      t.setText(content);
      return t;
    },

    renderResult(result, { expanded, isPartial }, theme, _context) {
      if (isPartial) return new Text(theme.fg("warning", "Running..."), 0, 0);
      const details = result.details as { code: number; stdout: string; stderr: string } | undefined;
      const stdout = details?.stdout?.trim() ?? "";
      const stderr = details?.stderr?.trim() ?? "";
      const exitCode = details?.code ?? 0;
      let text = "";
      if (exitCode !== 0) {
        text += theme.fg("error", `exit ${exitCode}`);
      } else if (!stdout && !stderr) {
        text += theme.fg("muted", "(no output)");
      } else {
        const preview = (stdout || stderr).split("\n")[0];
        text += theme.fg("success", preview);
      }
      if (expanded) {
        if (stdout) text += "\n" + theme.fg("dim", stdout);
        if (stderr) text += "\n" + theme.fg("error", stderr);
      }
      return new Text(text, 0, 0);
    },

    async execute(toolCallId, params, signal) {
      const result = await pi.exec("nrepl-send", params.args, { signal, timeout: 30000 });
      const parts: string[] = [];
      if (result.stdout.trim()) parts.push(result.stdout.trim());
      if (result.stderr.trim()) parts.push(`[stderr]\n${result.stderr.trim()}`);
      const text = parts.join("\n") || "(no output)";
      return {
        content: [{ type: "text", text }],
        details: { code: result.code, stdout: result.stdout, stderr: result.stderr },
      };
    },
  });
}
