package com.edwardnoaland.scryer.scan.inspect

internal fun expandProperties(value: String, properties: Map<String, String>): String {
    val reference = Regex("\\$\\{([\\w.]+)}|\\$([A-Za-z_]\\w*)")
    var expanded = value
    repeat(16) {
        val next = reference.replace(expanded) { match ->
            properties[match.groupValues[1].ifEmpty { match.groupValues[2] }] ?: match.value
        }
        if (next == expanded) return expanded
        expanded = next
    }
    return expanded // Cycles remain expressions, never invented versions.
}
