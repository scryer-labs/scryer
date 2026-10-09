package com.edwardnoaland.scryer.analyze.evidence

import com.edwardnoaland.scryer.analyze.SymbolId
import org.w3c.dom.Element
import java.nio.file.Path
import javax.xml.XMLConstants
import javax.xml.parsers.DocumentBuilderFactory
import org.xml.sax.InputSource
import org.xml.sax.SAXParseException
import org.xml.sax.helpers.DefaultHandler
import java.io.StringReader

internal object EvidenceXml {
    private fun root(path: Path): Element {
        val factory = DocumentBuilderFactory.newInstance().apply {
            setFeature(XMLConstants.FEATURE_SECURE_PROCESSING, true)
            setFeature("http://xml.org/sax/features/external-general-entities", false)
            setFeature("http://xml.org/sax/features/external-parameter-entities", false)
            setFeature("http://apache.org/xml/features/nonvalidating/load-external-dtd", false)
            setAttribute(XMLConstants.ACCESS_EXTERNAL_DTD, "")
            setAttribute(XMLConstants.ACCESS_EXTERNAL_SCHEMA, "")
            isXIncludeAware = false
            isExpandEntityReferences = false
        }
        val builder = factory.newDocumentBuilder().apply {
            setEntityResolver { _, _ -> InputSource(StringReader("")) }
            setErrorHandler(object : DefaultHandler() {
                override fun error(exception: SAXParseException) { throw exception }
                override fun fatalError(exception: SAXParseException) { throw exception }
            })
        }
        return path.toFile().inputStream().use { builder.parse(it).documentElement }
    }

    fun tests(artifact: EvidenceArtifact): TestReport {
        val document = root(artifact.retained)
        require(document.tagName in setOf("testsuite", "testsuites")) { "Not a JUnit test report" }
        val nodes = document.getElementsByTagName("testcase")
        val cases = (0 until nodes.length).map { index ->
            val node = nodes.item(index) as Element
            val status = when {
                node.children("error").isNotEmpty() -> TestCaseStatus.ERROR
                node.children("failure").isNotEmpty() || node.children("flakyFailure").isNotEmpty() || node.children("rerunFailure").isNotEmpty() -> TestCaseStatus.FAILED
                node.children("skipped").isNotEmpty() -> TestCaseStatus.SKIPPED
                else -> TestCaseStatus.PASSED
            }
            TestCaseRecord(node.getAttribute("classname").ifBlank { null }, node.getAttribute("name"), status, node.getAttribute("time").toDoubleOrNull())
        }
        return TestReport(artifact, cases)
    }

    fun coverage(artifact: EvidenceArtifact): CoverageDataset {
        val document = root(artifact.retained)
        require(document.tagName == "report") { "Not a JaCoCo XML report" }
        val nodes = document.getElementsByTagName("class")
        val classes = (0 until nodes.length).map { index ->
            val node = nodes.item(index) as Element
            val owner = node.getAttribute("name").replace('/', '.')
            val methods = node.children("method").map { method ->
                fun counter(kind: String): CoverageCounter {
                    val counter = method.children("counter").singleOrNull { it.getAttribute("type") == kind }
                    if (counter == null) {
                        require(kind != "INSTRUCTION") { "Missing method instruction counter" }
                        return CoverageCounter(0, 0)
                    }
                    val missed = counter.getAttribute("missed").toInt()
                    val covered = counter.getAttribute("covered").toInt()
                    require(missed >= 0 && covered >= 0) { "Invalid coverage counter" }
                    return CoverageCounter(missed, covered)
                }
                MethodCoverage(SymbolId(owner, method.getAttribute("name"), method.getAttribute("desc")),
                    method.getAttribute("line").toIntOrNull(), counter("INSTRUCTION"), counter("BRANCH"), counter("LINE"))
            }
            ClassCoverage(owner, null, ClassDataMatch.XML_UNVERIFIED, methods)
        }
        return CoverageDataset(artifact, "JACOCO_XML", classes, listOf("XML has no class ID validation; counters are not attributed to individual tests."))
    }

    private fun Element.children(name: String): List<Element> = (0 until childNodes.length).mapNotNull {
        (childNodes.item(it) as? Element)?.takeIf { element -> element.tagName == name }
    }
}
