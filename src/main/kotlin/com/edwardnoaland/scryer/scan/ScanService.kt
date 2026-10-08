package com.edwardnoaland.scryer.scan

import com.edwardnoaland.scryer.scan.inspect.RepositoryScanner
import com.edwardnoaland.scryer.scan.model.RepositoryFacts
import com.edwardnoaland.scryer.scan.resolve.DependencyResolver
import java.nio.file.Path

/** Coordinates collection; neither inspection nor resolution knows about CLI rendering. */
class ScanService(
    private val scanner: RepositoryScanner = RepositoryScanner(),
    private val resolver: DependencyResolver = DependencyResolver(),
) {
    fun scan(path: Path, staticOnly: Boolean = false): RepositoryFacts {
        val declaredFacts = scanner.scan(path)
        if (staticOnly) {
            return declaredFacts
        }

        val resolution = resolver.resolve(declaredFacts)
        return withResolution(declaredFacts, resolution)
    }
}
