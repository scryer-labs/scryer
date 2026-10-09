package com.edwardnoaland.scryer.cli.output

import com.edwardnoaland.scryer.scan.model.RepositoryFacts
import java.nio.file.AtomicMoveNotSupportedException
import java.nio.file.Files
import java.nio.file.Path
import java.nio.file.StandardCopyOption.ATOMIC_MOVE
import java.nio.file.StandardCopyOption.REPLACE_EXISTING
import kotlin.io.path.isDirectory

internal fun writeMarkdownReport(facts: RepositoryFacts, destination: Path, includeTree: Boolean) {
    writeReportFile(destination) { MarkdownReport().render(facts, includeTree) }
}

internal fun writeReportFile(path: Path, render: () -> String) {
    val destination = path.toAbsolutePath().normalize()
    require(!destination.isDirectory()) { "Output path is a directory: $destination" }
    val content = render()
    Files.createDirectories(destination.parent)
    val temporary = Files.createTempFile(destination.parent, ".scryer-report-", ".tmp")
    try {
        Files.writeString(temporary, content, Charsets.UTF_8)
        try {
            Files.move(temporary, destination, ATOMIC_MOVE, REPLACE_EXISTING)
        } catch (_: AtomicMoveNotSupportedException) {
            Files.move(temporary, destination, REPLACE_EXISTING)
        }
    } finally {
        Files.deleteIfExists(temporary)
    }
}
