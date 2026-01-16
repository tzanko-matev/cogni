import { defineConfig } from "vitest/config";

/**
 * Vitest configuration for the report UI.
 */
export default defineConfig({
  test: {
    environment: "node",
    testTimeout: 1000,
    include: ["src/report/tests/**/*.test.ts"],
  },
});
