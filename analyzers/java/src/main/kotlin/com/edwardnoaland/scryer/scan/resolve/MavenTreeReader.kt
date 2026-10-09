package com.edwardnoaland.scryer.scan.resolve

import com.edwardnoaland.scryer.scan.model.ConfigurationGraph
import com.edwardnoaland.scryer.scan.model.GraphEdge
import com.edwardnoaland.scryer.scan.model.GraphNode
import com.fasterxml.jackson.databind.JsonNode

internal fun readMavenTree(root: JsonNode, module: String): ConfigurationGraph {
    val nodes = linkedMapOf<String, GraphNode>()
    val edges = mutableListOf<GraphEdge>()
    fun id(node: JsonNode) = listOf("groupId", "artifactId", "type", "classifier", "version").joinToString(":") { node[it]?.asText().orEmpty() }
    val rootId = "project:$module"
    nodes[rootId] = GraphNode(rootId, null, null, null, "project")
    fun traverse(node: JsonNode, parent: String, depth: Int) {
        val nodeId = id(node)
        nodes[nodeId] = GraphNode(nodeId, node["groupId"]?.asText(), node["artifactId"]?.asText(), node["version"]?.asText(),
            type = node["type"]?.asText(), classifier = node["classifier"]?.asText(), optional = node["optional"]?.asBoolean())
        edges += GraphEdge(parent, nodeId, "${node["groupId"].asText()}:${node["artifactId"].asText()}", null,
            node["version"]?.asText(), depth == 0, conflict = null, forced = null, changedSelection = null, reasons = listOf("Maven selected tree entry; requested version/selection reason not supplied"), scope = node["scope"]?.asText())
        node["children"]?.forEach { traverse(it, nodeId, depth + 1) }
    }
    root["children"]?.forEach { traverse(it, rootId, 0) }
    return ConfigurationGraph(module, "Maven dependency tree (scope on nodes)", rootId, nodes.values.toList(), edges, "partial")
}
