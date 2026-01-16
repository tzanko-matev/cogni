import { describe, expect, it } from "vitest";
import { buildMetricPointsViewSQL, escapeSqlString } from "../sql";

describe("sql helpers", () => {
  it("escapes single quotes", () => {
    expect(escapeSqlString("o'reilly")).toBe("o''reilly");
  });

  it("builds metric points view with required filters", () => {
    const sql = buildMetricPointsViewSQL("tokens");
    expect(sql).toContain("v.metric = 'tokens'");
    expect(sql).toContain("v.status = 'ok'");
    expect(sql).toContain("v.value IS NOT NULL");
  });
});
