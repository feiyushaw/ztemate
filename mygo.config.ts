import { defineConfig } from "mygo-cli";
export default defineConfig({
  name: "ZTEMate",
  identifier: "io.github.feiyushaw.ztemate",
  version: "0.1.1",
  devUrl: "http://127.0.0.1:5173",
  devCommand: "bun run dev:web",
  buildCommand: "bun run build:web",
  frontendDist: "dist",
  bindings: "src/mygo.generated.ts",
  out: "build",
  macos: {
    signingIdentity: process.env.MACOS_SIGNING_IDENTITY || "-",
    infoPlist: { LSUIElement: true, NSLocalNetworkUsageDescription: "ZTEMate 需要访问你指定的局域网路由器，以读取网络状态和短信。" },
  },
});
