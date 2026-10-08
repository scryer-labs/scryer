package com.edwardnoaland.scryer.analyze.evidence

import java.nio.file.FileVisitResult
import java.nio.file.Files
import java.nio.file.Path
import java.nio.file.SimpleFileVisitor
import java.nio.file.attribute.BasicFileAttributes
import java.security.MessageDigest

internal enum class ArtifactKind { TEST_XML, JACOCO_EXEC, JACOCO_XML, CLASS }
internal data class ArtifactStamp(val kind: ArtifactKind, val hash: String, val modified: String, val fileKey: String?)
internal object ArtifactInventory {
    fun capture(root: Path): Map<Path, ArtifactStamp> {
        val result = linkedMapOf<Path, ArtifactStamp>()
        Files.walkFileTree(root, object : SimpleFileVisitor<Path>() {
            override fun preVisitDirectory(dir: Path, attrs: BasicFileAttributes): FileVisitResult =
                if (dir != root && dir.fileName.toString() in EXCLUDED) FileVisitResult.SKIP_SUBTREE else FileVisitResult.CONTINUE
            override fun visitFile(file: Path, attrs: BasicFileAttributes): FileVisitResult {
                if (!attrs.isRegularFile || Files.isSymbolicLink(file)) return FileVisitResult.CONTINUE
                val relative = root.relativize(file).toString().replace('\\', '/')
                val kind = kind(relative) ?: return FileVisitResult.CONTINUE
                result[file] = ArtifactStamp(kind, hash(file), attrs.lastModifiedTime().toString(), attrs.fileKey()?.toString())
                return FileVisitResult.CONTINUE
            }
        })
        return result
    }

    fun hash(path: Path): String {
        val digest = MessageDigest.getInstance("SHA-256")
        Files.newInputStream(path).use { stream ->
            val buffer = ByteArray(8192)
            while (true) {
                val count = stream.read(buffer)
                if (count < 0) break
                digest.update(buffer, 0, count)
            }
        }
        return digest.digest().joinToString("") { "%02x".format(it) }
    }

    private fun kind(relative: String): ArtifactKind? {
        val path = "/$relative"
        return when {
            relative.endsWith(".xml") && ("/build/test-results/" in path || "/target/surefire-reports/" in path || "/target/failsafe-reports/" in path) && relative.substringAfterLast('/').startsWith("TEST-") -> ArtifactKind.TEST_XML
            relative.endsWith(".exec") && ("/build/" in path || "/target/" in path) -> ArtifactKind.JACOCO_EXEC
            relative.endsWith(".xml") && ("/build/reports/jacoco/" in path || Regex("/target/site/jacoco[^/]*/").containsMatchIn(path)) -> ArtifactKind.JACOCO_XML
            relative.endsWith(".class") && ("/build/classes/java/main/" in path || "/build/classes/main/" in path || "/target/classes/" in path) -> ArtifactKind.CLASS
            else -> null
        }
    }
    private val EXCLUDED = setOf(".git", ".gradle", ".tooling", "node_modules", "src")
}
