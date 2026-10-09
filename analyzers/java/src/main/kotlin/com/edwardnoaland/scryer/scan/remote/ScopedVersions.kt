package com.edwardnoaland.scryer.scan.remote

import com.edwardnoaland.scryer.scan.model.DependencyDeclaration
import com.edwardnoaland.scryer.scan.model.RepositoryFacts

/** A dependency's current version comes from its scope/configuration, never an unrelated module graph. */
internal fun scopedSelectedVersions(facts: RepositoryFacts, dependency: DependencyDeclaration): List<String> {
    return facts.resolution.configurations.filter { it.module == dependency.module }.flatMap { graph ->
        val maven = graph.configuration.startsWith("Maven dependency tree")
        val relevant = maven || graph.configuration == dependency.configuration ||
            dependency.configuration in graph.declarationConfigurations
        if (!relevant) return@flatMap emptyList()
        val scopedTargets = if (maven) graph.edges.filter { it.direct && it.scope == dependency.configuration }.mapNotNull { it.to }.toSet() else null
        graph.nodes.filter { node ->
            node.group == dependency.group && node.artifact == dependency.artifact &&
                (scopedTargets == null || node.id in scopedTargets) &&
                (dependency.classifier == null || dependency.classifier == node.classifier) &&
                (dependency.type == null || dependency.type == node.type)
        }.mapNotNull { it.version }
    }.distinct().sorted()
}
