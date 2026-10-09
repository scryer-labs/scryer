package com.edwardnoaland.scryer.analyze.model

import org.w3c.dom.Element
import java.nio.file.Path
import javax.xml.parsers.DocumentBuilderFactory
import javax.xml.transform.TransformerFactory
import javax.xml.transform.dom.DOMSource
import javax.xml.transform.stream.StreamResult

/** Resolve external jars without requiring unpublished reactor artifacts or executing project plugins. */
internal fun writeExternalClasspathPom(project: Element, output: Path, reactor: Set<Pair<String, String>>) {
    fun Element.children() = (0 until childNodes.length).mapNotNull { childNodes.item(it) as? Element }
    fun Element.text(name: String) = children().firstOrNull { it.tagName == name }?.textContent?.trim().orEmpty()
    val document = DocumentBuilderFactory.newInstance().newDocumentBuilder().newDocument()
    val root = document.createElement("project")
    document.appendChild(root)
    val retained = setOf("modelVersion", "groupId", "artifactId", "version", "properties", "dependencyManagement", "dependencies", "repositories", "pluginRepositories")
    for (element in project.children().filter { it.tagName in retained }) {
        val copy = document.importNode(element, true) as Element
        if (copy.tagName == "dependencies") {
            for (dependency in copy.children()) {
                if ((dependency.text("groupId") to dependency.text("artifactId")) in reactor) copy.removeChild(dependency)
            }
        }
        root.appendChild(copy)
    }
    TransformerFactory.newInstance().newTransformer().transform(DOMSource(document), StreamResult(output.toFile()))
}
