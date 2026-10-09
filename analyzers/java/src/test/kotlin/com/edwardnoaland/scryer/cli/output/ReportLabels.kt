package com.edwardnoaland.scryer.cli.output

/** Presentation only: never use compact labels as symbol identities. */
internal fun compactSignature(signature: String): String {
    val separator = signature.indexOf('#')
    if (separator < 0) return signature
    val owner = signature.substring(0, separator).substringAfterLast('.')
    return owner + signature.substring(separator)
}

internal fun signatureLabels(signatures: Collection<String>): Map<String, String> {
    val groups = signatures.distinct().groupBy(::compactSignature)
    return groups.flatMap { (label, originals) ->
        originals.map { original -> original to if (originals.size == 1) label else original }
    }.toMap()
}

internal fun compactSource(path: String?): String = path?.substringAfterLast('/') ?: "unknown"
