package com.edwardnoaland.scryer.cli.output

import com.edwardnoaland.scryer.analyze.report.AnalyzeReport

/** Inline assets and script-safe JSON keep reports portable and usable under file://. */
internal fun renderAnalyzeHtml(report: AnalyzeReport): String {
    val template = reportResource("report.html")
        .replace("__SCRYER_STYLE__", reportResource("report.css"))
        .replace("__SCRYER_APP__", reportResource("report.js"))
        .replace("__SCRYER_GRAPH__", reportResource("graph-model.js") + "\n" + reportResource("graph.js"))
    val data = renderAnalyzeJson(report).replace("&", "\\u0026").replace("<", "\\u003c").replace(">", "\\u003e")
        .replace("\u2028", "\\u2028").replace("\u2029", "\\u2029")
    return template.replace("__SCRYER_DATA__", data)
}

private fun reportResource(name: String): String = checkNotNull(AnalyzeReport::class.java.getResourceAsStream("/analyze-report/$name")) {
    "Missing bundled report asset: $name"
}.bufferedReader(Charsets.UTF_8).use { it.readText() }
