export default [
  {
    files: ["js/**/*.js"],
    languageOptions: {
      ecmaVersion: 2020,
      sourceType: "module",
      globals: {
        Blob: "readonly",
        console: "readonly",
        document: "readonly",
        fetch: "readonly",
        FileReader: "readonly",
        Image: "readonly",
        L: "readonly",
        m: "readonly",
        performance: "readonly",
        setTimeout: "readonly",
        THREE: "readonly",
        URL: "readonly",
        WebSocket: "readonly",
        window: "readonly",
      },
    },
    rules: {
      "no-undef": "error",
      "no-unused-vars": ["error", { args: "none" }],
    },
  },
];
