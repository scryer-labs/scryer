package com.edwardnoaland.scryer.scan

import com.edwardnoaland.scryer.repository.BuildTool

import com.edwardnoaland.scryer.scan.inspect.RepositoryScanner
import com.edwardnoaland.scryer.scan.model.RepositoryFacts
import com.edwardnoaland.scryer.scan.resolve.DependencyResolver
import java.nio.file.Path

/** Coordinates collection; neither inspection nor resolution knows about CLI rendering. */
class ScanService(
    private val scanner: RepositoryScanner = RepositoryScanner(),
    private val resolver: DependencyResolver = DependencyResolver(),
) {
    fun scan(path: Path, staticOnly: Boolean = false, buildTool: BuildTool? = null): RepositoryFacts {
        val declaredFacts = scanner.scan(path, buildTool)
        if (staticOnly) {
            return declaredFacts
        }

        val selectedBuilds = declaredFacts.builds.filter { buildTool == null || it.tool == buildTool.displayName }
        val resolution = resolver.resolve(declaredFacts.copy(builds = selectedBuilds))
        return withResolution(declaredFacts, resolution)
    }
}
