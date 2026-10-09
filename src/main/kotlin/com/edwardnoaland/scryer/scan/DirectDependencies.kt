package com.edwardnoaland.scryer.scan

import com.edwardnoaland.scryer.scan.model.DependencyDeclaration
import com.edwardnoaland.scryer.scan.model.RepositoryFacts

internal fun directDependencies(facts: RepositoryFacts): List<DependencyDeclaration> {
    val literal = facts.dependencies.filter { it.kind != "managed declaration" }
    val evaluated = facts.resolution.projects.flatMap { project -> project.declaredDependencies.filter { it.kind == "dependency" && it.group != null }.map { dep ->
        DependencyDeclaration(dep.configuration, "${dep.group}:${dep.artifact}", dep.version, "evaluated Gradle model", module = project.id)
    } }
    // Preserve original declared versions/provenance; evaluated declarations fill unsupported DSL gaps.
    val missing = evaluated.filter { candidate -> literal.none { it.module == candidate.module && it.configuration == candidate.configuration &&
        it.notation == candidate.notation && it.version == candidate.version } }
    return (literal + missing).distinctBy { "${it.module}|${it.configuration}|${it.notation}|${it.version}|${it.type}|${it.classifier}" }
}
