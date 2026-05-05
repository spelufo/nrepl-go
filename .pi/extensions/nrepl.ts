import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";
import { Type } from "typebox";


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
