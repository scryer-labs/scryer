package com.edwardnoaland.scryer.scan.model

data class RemoteVersion(val module: String, val configuration: String, val coordinate: String,
    val current: String?, val currentSource: String, val latest: String?, val status: String,
    val metadataUrl: String?, val note: String? = null)
data class RemoteVersions(val checkedAt: String, val source: String, val dependencies: List<RemoteVersion>)
