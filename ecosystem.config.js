module.exports = {
  apps: [
    {
      name: "master",
      script: "go",
      args: "run ./cmd/master",
      watch: false,
      env: {
        BRAZIER_PORT: "9000",
        BRAZIER_HTTP_PORT: "8080",
      },
    },
    {
      name: "web",
      script: "npm",
      args: "run dev",
      cwd: "./web",
      watch: false,
    },
  ],
};
