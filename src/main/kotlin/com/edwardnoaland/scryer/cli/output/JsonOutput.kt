package com.edwardnoaland.scryer.cli.output

import com.edwardnoaland.scryer.scan.model.RepositoryFacts
import com.edwardnoaland.scryer.serialization.jsonMapper
import com.fasterxml.jackson.databind.node.ObjectNode
import java.io.PrintStream

internal fun renderJson(facts: RepositoryFacts, out: PrintStream) {
    val json = jsonMapper.valueToTree<ObjectNode>(facts)
    json.put("root", facts.root.toString())
    out.println(jsonMapper.writerWithDefaultPrettyPrinter().writeValueAsString(json))
}
