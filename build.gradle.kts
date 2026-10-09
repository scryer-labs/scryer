plugins {
    kotlin("jvm") version "2.2.21"
    application
}

group = "com.edwardnoaland.scryer"
version = "0.1.0"

repositories { mavenCentral() }
kotlin {
    jvmToolchain(21)
    sourceSets {
        named("main") { kotlin.setSrcDirs(listOf("analyzers/java/src/main/kotlin")) }
        named("test") { kotlin.setSrcDirs(listOf("analyzers/java/src/test/kotlin")) }
    }
}
application {
    applicationName = "scryer-java"
    mainClass.set("com.edwardnoaland.scryer.worker.WorkerKt")
}
sourceSets {
    named("main") { resources.setSrcDirs(listOf("analyzers/java/src/main/resources")) }
    named("test") { resources.setSrcDirs(listOf("analyzers/java/src/test/resources")) }
}
distributions { named("main") { distributionBaseName = "scryer-java" } }
tasks.named<JavaExec>("run") { standardInput = System.`in` }
tasks.processTestResources {
    from("internal/report/assets") { into("analyze-report") }
}

dependencies {
    testImplementation("com.github.ajalt.mordant:mordant-core:3.0.2")
    implementation("com.fasterxml.jackson.module:jackson-module-kotlin:2.18.3")
    implementation("org.apache.maven:maven-artifact:3.9.9")
    implementation("org.jacoco:org.jacoco.core:0.8.14")
    testImplementation(platform("org.junit:junit-bom:5.11.4"))
    testImplementation(kotlin("test-junit5"))
    testRuntimeOnly("org.junit.platform:junit-platform-launcher")
}

tasks.test { useJUnitPlatform() }
