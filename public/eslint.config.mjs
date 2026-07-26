export default [
  {
    ignores: ["js/bundle.js", "js/bundle-stats.js", "js/lib/**"],
  },
  {
    files: ["js/**/*.js"],
    languageOptions: {
      ecmaVersion: 2020,
      sourceType: "module",
      globals: {
        Blob: "readonly",
        clearTimeout: "readonly",
        console: "readonly",
        document: "readonly",
        fetch: "readonly",
        FileReader: "readonly",
        Image: "readonly",
        L: "readonly",
        localStorage: "readonly",
        m: "readonly",
        performance: "readonly",
        setInterval: "readonly",
        setTimeout: "readonly",
        THREE: "readonly",
        URL: "readonly",
        WebSocket: "readonly",
        window: "readonly",
      },
    },
    rules: {
      "no-undef": "error",
      "no-unused-vars": ["error", { args: "none", caughtErrors: "none" }],
    },
  },
];
