package com.edwardnoaland.scryer.scan.model

data class ToolUpdate(
    val tool: String, val channel: String, val current: String?, val currentSource: String,
    val latest: String?, val status: String, val metadataUrl: String?, val note: String?,
)
data class ToolUpdates(val checkedAt: String, val tools: List<ToolUpdate>, val notes: List<String>)

