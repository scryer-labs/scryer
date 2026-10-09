package com.edwardnoaland.scryer.cli.output

import com.edwardnoaland.scryer.analyze.report.AnalyzeReport
import com.edwardnoaland.scryer.serialization.jsonMapper

internal fun renderAnalyzeJson(report: AnalyzeReport): String = jsonMapper.writerWithDefaultPrettyPrinter().writeValueAsString(report) + "\n"
