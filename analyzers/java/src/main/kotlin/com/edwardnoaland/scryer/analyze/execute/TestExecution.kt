package com.edwardnoaland.scryer.analyze.execute

import java.nio.file.Path

/** Command exit evidence only; success does not prove that tests executed or covered code. */
enum class TestRunStatus { SUCCEEDED, FAILED, TIMED_OUT, UNAVAILABLE, SKIPPED, SNAPSHOT_CHANGED }
data class TestExecution(
    val afterSha: String,
    val status: TestRunStatus,
    val command: List<String> = emptyList(),
    val javaHome: String? = null,
    val exitCode: Int? = null,
    val durationMillis: Long = 0,
    val log: Path? = null,
    val notes: List<String> = emptyList(),
)
internal data class TestRunPlan(val command: List<String>, val javaHome: Path, val environment: Map<String, String>)
