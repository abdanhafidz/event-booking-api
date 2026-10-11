"""Compare a load test report with perf/thresholds.json.

Usage: python scripts/perf/check_thresholds.py <scenario> <report.json> <thresholds.json>

<report.json> is either BlazeMeter's aggregate report (aggregatereport/data) or
JMeter's dashboard statistics.json, so local and BlazeMeter runs share one gate.
Prints a Markdown table. Exits 1 when a strict gate fails, or when a target
fails and the scenario has "enforce": true.
"""
import json
import sys


def normalise(report):
    """Return {label: {samples, errors, p95, p99, rps}}; the overall row is "ALL"."""
    rows = {}
    if isinstance(report, dict) and "result" in report:  # BlazeMeter
        for r in report["result"]:
            rows[r["labelName"]] = {
                "samples": r.get("samples", 0),
                "errors": r.get("errorsCount", 0),
                "p95": r.get("95line"),
                "p99": r.get("99line"),
                "rps": r.get("avgThroughput"),
            }
    else:  # JMeter statistics.json
        for label, r in report.items():
            rows["ALL" if label == "Total" else label] = {
                "samples": r.get("sampleCount", 0),
                "errors": r.get("errorCount", 0),
                "p95": r.get("pct2ResTime"),
                "p99": r.get("pct3ResTime"),
                "rps": r.get("throughput"),
            }
    return rows


def evaluate(rows, gates):
    """Yield (check, limit, actual, ok) for each gate."""
    overall = rows.get("ALL", {"samples": 0, "errors": 0})
    if "max_error_pct" in gates:
        pct = 100.0 * overall["errors"] / overall["samples"] if overall["samples"] else 100.0
        yield "unexpected errors %", f"<= {gates['max_error_pct']}", round(pct, 3), pct <= gates["max_error_pct"]
    if "min_rps" in gates:
        rps = overall.get("rps") or 0
        yield "throughput req/s", f">= {gates['min_rps']}", round(rps, 1), rps >= gates["min_rps"]
    for pct_key, name in (("p95_ms", "p95"), ("p99_ms", "p99")):
        for label, limit in gates.get(pct_key, {}).items():
            value = rows.get(label, {}).get(name)
            ok = value is not None and value <= limit
            yield f"{name} {label} (ms)", f"<= {limit}", value, ok
    for label, want in gates.get("expect_samples", {}).items():
        got = rows.get(label, {}).get("samples", 0)
        yield f"samples {label}", f"= {want}", got, got == want


def main():
    scenario, report_path, thresholds_path = sys.argv[1:4]
    with open(report_path, encoding="utf-8") as f:
        rows = normalise(json.load(f))
    with open(thresholds_path, encoding="utf-8") as f:
        config = json.load(f)[scenario]

    enforce = config.get("enforce", False)
    failed = False
    print(f"### Thresholds for {scenario} (targets {'enforced' if enforce else 'reported only until calibrated'})\n")
    print("| Gate | Check | Limit | Actual | Result |")
    print("| --- | --- | --- | --- | --- |")
    for kind, gates in (("strict", config.get("strict", {})), ("target", config.get("targets", {}))):
        for check, limit, actual, ok in evaluate(rows, gates):
            blocking = kind == "strict" or enforce
            result = "pass" if ok else ("**FAIL**" if blocking else "warn")
            failed |= (not ok) and blocking
            print(f"| {kind} | {check} | {limit} | {actual} | {result} |")
    print()
    sys.exit(1 if failed else 0)


if __name__ == "__main__":
    main()
