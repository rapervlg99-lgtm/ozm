import { describe, expect, it } from "vitest";
import {
  availableCoatingTypes,
  coatingFireLimitRange,
  coatingFitHint,
  coatingFitsRow,
  coatingTypeChoices,
  isCoatingAvailableForFireLimit,
  parseFireLimitMinutes
} from "./coatingTypes";

describe("coatingTypes", () => {
  it("парсит минуты из предела ОС", () => {
    expect(parseFireLimitMinutes("R90")).toBe(90);
    expect(parseFireLimitMinutes("R 120")).toBe(120);
    expect(parseFireLimitMinutes("60")).toBe(60);
    expect(parseFireLimitMinutes("")).toBeNull();
  });

  it("без R доступны все типы", () => {
    expect(availableCoatingTypes("")).toHaveLength(5);
    expect(availableCoatingTypes()).toHaveLength(5);
  });

  it("TAIKOR Epoxy/Graphite только до R120", () => {
    expect(isCoatingAvailableForFireLimit("TAIKOR FP Epoxy", "R60")).toBe(true);
    expect(isCoatingAvailableForFireLimit("TAIKOR FP Graphite", "R120")).toBe(
      true
    );
    expect(isCoatingAvailableForFireLimit("TAIKOR FP Epoxy", "R150")).toBe(
      false
    );
  });

  it("Extra+Graphite только R90 и R120", () => {
    expect(
      isCoatingAvailableForFireLimit(
        "TAIKOR FP Extra + TAIKOR FP Graphite",
        "R90"
      )
    ).toBe(true);
    expect(
      isCoatingAvailableForFireLimit(
        "TAIKOR FP Extra + TAIKOR FP Graphite",
        "R60"
      )
    ).toBe(false);
    expect(
      isCoatingAvailableForFireLimit(
        "TAIKOR FP Extra + TAIKOR FP Graphite",
        "R150"
      )
    ).toBe(false);
  });

  it("ОЗМ и АКЗ доступны при любом R", () => {
    for (const r of ["R15", "R90", "R240"]) {
      expect(isCoatingAvailableForFireLimit("ТЕХНО ОЗМ", r)).toBe(true);
      expect(isCoatingAvailableForFireLimit("АКЗ", r)).toBe(true);
    }
  });

  it("choices сохраняет неизвестное текущее значение", () => {
    const list = coatingTypeChoices("СГК-1", "R150");
    expect(list[0]).toBe("СГК-1");
    expect(list).toContain("ТЕХНО ОЗМ");
    expect(list).not.toContain("TAIKOR FP Epoxy");
  });

  it("подсказка объясняет, почему покрытие не подходит к пределу ОС", () => {
    expect(coatingFitHint("TAIKOR FP Graphite", "R180")).toBe(
      "TAIKOR FP Graphite не подходит для R180 — это покрытие бывает при R15…R120"
    );
    expect(coatingFitHint("TAIKOR FP Extra + TAIKOR FP Graphite", "R45")).toBe(
      "TAIKOR FP Extra + TAIKOR FP Graphite не подходит для R45 — это покрытие бывает при R90…R120"
    );
    // совместимые сочетания молчат
    expect(coatingFitHint("TAIKOR FP Graphite", "R120")).toBe("");
    expect(coatingFitHint("ТЕХНО ОЗМ", "R240")).toBe("");
    expect(coatingFitHint("АКЗ", "R240")).toBe("");
    expect(coatingFitHint("", "R180")).toBe("");
  });

  it("пределы применимости покрытий", () => {
    expect(coatingFireLimitRange("TAIKOR FP Epoxy")).toBe("R15…R120");
    expect(coatingFireLimitRange("TAIKOR FP Extra + TAIKOR FP Graphite")).toBe("R90…R120");
    expect(coatingFireLimitRange("ТЕХНО ОЗМ")).toBe("");
  });

  it("проверка строки: покрытие и её предел ОС", () => {
    expect(coatingFitsRow({ coatingType: "TAIKOR FP Graphite", fireLimit: "R180" })).toBe(false)
    expect(coatingFitsRow({ coatingType: "TAIKOR FP Graphite", fireLimit: "R120" })).toBe(true)
    expect(coatingFitsRow({ coatingType: "", fireLimit: "R180" })).toBe(true)
  })
});
