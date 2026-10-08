package com.edwardnoaland.scryer.scan.inspect

import com.edwardnoaland.scryer.scan.model.DependencyDeclaration
import com.edwardnoaland.scryer.scan.model.LanguageFacts
import com.edwardnoaland.scryer.scan.model.ModuleFacts
import com.edwardnoaland.scryer.scan.model.PluginDeclaration
import com.edwardnoaland.scryer.scan.model.SignalFact
import com.edwardnoaland.scryer.scan.model.SourceFacts
import com.edwardnoaland.scryer.scan.model.TestingFacts
import com.edwardnoaland.scryer.scan.model.VerificationFacts
import java.nio.file.Files
import java.nio.file.Path
import kotlin.io.path.*

internal data class Inspection(
    val language: LanguageFacts, val sources: SourceFacts, val testing: TestingFacts,
    val signals: List<SignalFact>, val verification: VerificationFacts,
    val repositories: List<String>, val customBuildFiles: List<String>, val frameworks: List<String>,
)

internal fun inspectMetadata(
    root: Path,
    modules: List<ModuleFacts>,
    dependencies: List<DependencyDeclaration>,
    plugins: List<PluginDeclaration>,
): Inspection {
    val excluded = setOf(".git", ".gradle", ".tooling", ".kotlin", ".idea", "node_modules", "target", "build")
    val collected = mutableListOf<Path>()
    Files.walkFileTree(root, object : java.nio.file.SimpleFileVisitor<Path>() {
        override fun preVisitDirectory(dir: Path, attrs: java.nio.file.attribute.BasicFileAttributes): java.nio.file.FileVisitResult =
            if (dir != root && dir.fileName.toString() in excluded) java.nio.file.FileVisitResult.SKIP_SUBTREE else java.nio.file.FileVisitResult.CONTINUE
        override fun visitFile(file: Path, attrs: java.nio.file.attribute.BasicFileAttributes): java.nio.file.FileVisitResult {
            if (attrs.isRegularFile && !attrs.isSymbolicLink) collected.add(file)
            return java.nio.file.FileVisitResult.CONTINUE
        }
    })
    val paths = collected.sortedBy { it.toString() }
    val sourceFiles = paths.filter { it.extension in setOf("java", "kt", "groovy") }
    val buildFiles = modules.map { root.resolve(it.directory).resolve(it.definition) }.filter { it.isRegularFile() }
    val buildTexts = buildFiles.map { it.readText() }
    val buildText = buildTexts.joinToString("\n")
    fun rel(p: Path) = root.relativize(p).toString().replace('\\', '/')
    fun javaVersion(value: String) = value.trim().trim('"', '\'').removePrefix("JavaVersion.VERSION_").replace('_', '.').removePrefix("1.")
    fun matches(pattern: String) = Regex(pattern).findAll(buildText).map { javaVersion(it.groupValues[1]) }.distinct().toList()
    val pomProperties = buildFiles.filter { it.fileName.toString() == "pom.xml" }.map { readPom(it) }
    fun mavenProperty(name: String) = pomProperties.mapNotNull { it.child("properties")?.text(name) }.map(::javaVersion)
    val language = LanguageFacts(
        sourceVersions = (matches("sourceCompatibility\\s*=\\s*(?:JavaVersion.VERSION_)?['\"]?([0-9._]+)") + mavenProperty("maven.compiler.source") + mavenProperty("maven.compiler.release") + mavenProperty("java.version")).distinct(),
        targetVersions = (matches("targetCompatibility\\s*=\\s*(?:JavaVersion.VERSION_)?['\"]?([0-9._]+)") + mavenProperty("maven.compiler.target") + mavenProperty("maven.compiler.release")).distinct(),
        toolchainVersions = matches("(?:JavaLanguageVersion\\.of|jvmToolchain)\\s*\\(\\s*([0-9]+)"),
        previewSignal = "--enable-preview" in buildText,
        sourceLanguages = sourceFiles.map { file ->
            when (file.extension) {
                "java" -> "Java"
                "kt" -> "Kotlin"
                else -> "Groovy"
            }
        }.distinct().sorted(),
        jpmsDescriptors = sourceFiles.filter { it.fileName.toString() == "module-info.java" }.map(::rel),
    )
    fun sourceRoot(path: Path): String? {
        val parts = rel(path).split('/')
        val src = parts.indexOf("src")
        return if (src >= 0 && parts.size > src + 3) parts.take(src + 3).joinToString("/") else null
    }
    val sourceRoots = sourceFiles.mapNotNull(::sourceRoot).distinct().sorted()
    val tests = sourceFiles.filter { file ->
        val relativePath = rel(file)
        relativePath.contains("/test/") || relativePath.contains("/integrationTest/") || relativePath.contains("/it/")
    }
    val integration = tests.filter { Regex("(?:IntegrationTest|IT)\\.(java|kt|groovy)$").containsMatchIn(it.fileName.toString()) || rel(it).contains("/integrationTest/") || rel(it).contains("/it/") }
    val generatedRoots = modules.flatMap { module ->
        listOf("build/generated", "target/generated-sources", "target/generated-test-sources", "src/generated")
            .map { root.resolve(module.directory).resolve(it) }.filter { it.isDirectory() }.map(::rel)
    }
    val sources = SourceFacts(sourceRoots.filter { "/main/" in it }, sourceRoots.filter { "/test/" in it },
        sourceRoots.filter { "/integrationTest/" in it || "/it/" in it }, generatedRoots,
        sourceFiles.count { it.extension == "java" })
    // These are lexical signals, not an AST/call graph or proof that a construct executes.
    val content = sourceFiles.associateWith { it.readText().replace(Regex("(?s)/\\*.*?\\*/"), "").replace(Regex("(?m)//[^\\n]*"), "") }
    val signalPatterns = linkedMapOf(
        "javax usage" to "\\bjavax\\.", "jakarta usage" to "\\bjakarta\\.", "Lombok" to "\\blombok\\.",
        "Reflection" to "Class\\.forName|\\.get(?:Declared)?Method\\s*\\(|java\\.lang\\.reflect", "Class.forName" to "Class\\.forName\\s*\\(",
        "Dynamic class loading" to "ClassLoader|\\.loadClass\\s*\\(", "ServiceLoader" to "ServiceLoader",
        "Spring proxies" to "@(?:Transactional|Async|Cacheable)|ProxyFactory|@EnableAspectJAutoProxy",
        "JNI/native" to "System\\.load(?:Library)?\\s*\\(|\\bnative\\s+", "Serialization" to "\\bSerializable\\b|ObjectInputStream|ObjectOutputStream",
        "Bytecode manipulation" to "org\\.objectweb\\.asm|javassist|net\\.bytebuddy",
    )
    val signals = signalPatterns.map { (name, pattern) -> SignalFact(name, content.filterValues { Regex(pattern).containsMatchIn(it) }.keys.map(::rel)) }
    val coordinates = dependencies.map { it.notation } + plugins.map { it.id }
    val joined = coordinates.joinToString(" ")
    val frameworks = linkedMapOf("Spring Boot" to "spring-boot", "Spring Framework" to "org.springframework:spring-", "Spring Security" to "spring-security", "Hibernate/JPA" to "hibernate|persistence-api", "Servlet" to "servlet|starter-web\\b", "WebFlux" to "webflux", "Tomcat" to "tomcat", "Jetty" to "jetty")
        .filterValues { Regex(it).containsMatchIn(joined) }.keys.toList()
    val testFrameworks = mutableSetOf<String>()
    val testText = tests.joinToString("\n") { content[it].orEmpty() }
    if (Regex("org\\.junit\\.(?:Test|runner|Assert)|junit:junit").containsMatchIn(testText + joined)) testFrameworks += "JUnit 4"
    if ("org.junit.jupiter" in testText || "junit-jupiter" in joined || "test-junit5" in buildText) testFrameworks += "JUnit 5"
    if ("org.testng" in testText || "testng:testng" in joined) testFrameworks += "TestNG"
    val reports = modules.flatMap { module ->
        listOf("build/reports/jacoco", "build/reports/evidence", "target/site/jacoco").map { root.resolve(module.directory).resolve(it) }
            .filter { it.isDirectory() }.flatMap { directory -> Files.walk(directory).use { paths -> paths.filter { path ->
                path.isRegularFile() && !Files.isSymbolicLink(path) && (path.extension == "xml" || path.fileName.toString() == "index.html")
            }.map(::rel).toList() } }
    }.distinct().sorted()
    val tools = listOf("surefire", "failsafe", "jacoco", "checkstyle", "spotbugs", "pmd", "sonar").filter { it in buildText.lowercase() }
    val taskSignals = Regex("(?:tasks\\.(?:create|register)|task)\\s*\\(?\\s*['\"]?(\\w*[Tt]est\\w*)").findAll(buildText).map { it.groupValues[1] }.distinct().toList()
    val testing = TestingFacts(testFrameworks.toList(), coordinates.filter { "mockito" in it }, coordinates.filter { "assertj" in it || "hamcrest" in it },
        tests.count { it !in integration }, integration.size, disabledAnnotationSignals = tests.sumOf { Regex("@(?:Ignore|Disabled)\\b").findAll(content[it].orEmpty()).count() },
        jacocoConfigured = "jacoco" in tools, coverageReports = reports, testTasksAndPlugins = (tools.filter { it in listOf("surefire", "failsafe") } + taskSignals).distinct())
    val verification = inspectVerification(root, modules, paths, tools)
    val repositories = Regex("mavenCentral\\(\\)|google\\(\\)|jcenter\\(\\)|https?://[^\\s\"'<>]+").findAll(buildText).map { it.value }.distinct().toList()
    val custom = paths.filter { val r = rel(it); r.startsWith("buildSrc/") || r.startsWith("scripts/") || it.extension in listOf("gradle", "kts") }.map(::rel)
    return Inspection(language, sources, testing, signals, verification, repositories, custom, frameworks)
}
