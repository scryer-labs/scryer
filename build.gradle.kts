plugins {
    kotlin("jvm") version "2.2.21"
    application
}

group = "com.edwardnoaland.scryer"
version = "0.1.0"

repositories { mavenCentral() }
kotlin { jvmToolchain(21) }
application { mainClass.set("com.edwardnoaland.scryer.MainKt") }

dependencies {
    implementation("com.github.ajalt.mordant:mordant-core:3.0.2")
    implementation("com.fasterxml.jackson.module:jackson-module-kotlin:2.18.3")
    implementation("org.jacoco:org.jacoco.core:0.8.14")
    testImplementation(platform("org.junit:junit-bom:5.11.4"))
    testImplementation(kotlin("test-junit5"))
    testRuntimeOnly("org.junit.platform:junit-platform-launcher")
}

tasks.test { useJUnitPlatform() }
