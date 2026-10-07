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
    testImplementation(platform("org.junit:junit-bom:5.11.4"))
    testImplementation(kotlin("test-junit5"))
    testRuntimeOnly("org.junit.platform:junit-platform-launcher")
}

tasks.test { useJUnitPlatform() }
