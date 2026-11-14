"""Install, scan and upgrade official releases using an isolated Scryer home."""
import json
import os
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

cli = Path(sys.argv[1]).resolve()
fixtures = {
    "cpp": ("main.cpp", "int main() { return 0; }", "files"),
    "rust": ("main.rs", "fn main() {}", "files"),
    "ts-js": ("main.ts", "export function run() {}", "files"),
    "go": ("main.go", "package main\nfunc main() {}", "files"),
    "java": ("Main.java", "class Main {}", "dependencies"),
}

with tempfile.TemporaryDirectory(prefix="scryer-release-check-") as directory:
    root = Path(directory)
    environment = {
        key: value for key, value in os.environ.items()
        if not key.startswith("SCRYER_ENGINE_") and key not in ("SCRYER_ANALYZER_JAVA_HOME", "SCRYER_JAVA_CLASSPATH")
    }
    environment["SCRYER_HOME"] = str(root / "home")
    environment["JAVA_HOME"] = str(root / "missing-java")

    def run(*args):
        return json.loads(subprocess.check_output([cli, *args, "--json"], env=environment))

    install = run("engine", "install", *fixtures)
    assert install["status"] == "installed", install
    assert {item["engine"]["id"] for item in install["engines"]} == fixtures.keys()
    assert run("engine", "upgrade", *fixtures)["status"] == "up-to-date"
    for item in install["engines"]:
        engine = item["engine"]["id"]
        file, text, fact = fixtures[engine]
        assert item["sourcePath"].startswith("https://github.com/")
        old = root / ("old-" + engine)
        shutil.copytree(item["packageRoot"], old, ignore=shutil.ignore_patterns(".scryer-install.json"))
        manifest = old / "engine.json"
        data = json.loads(manifest.read_text())
        data["engine"]["version"] = "0.0.0-dev"
        manifest.write_text(json.dumps(data))
        run("engine", "uninstall", engine)
        run("engine", "install", engine, "--from", old)
        upgraded = run("engine", "upgrade", engine)
        assert upgraded["status"] == "upgraded" and upgraded["engines"][0]["engine"] == item["engine"], upgraded
        source = root / engine
        source.mkdir()
        (source / file).write_text(text + "\n")
        if engine == "java":
            (source / "pom.xml").write_text("<project><modelVersion>4.0.0</modelVersion><groupId>test</groupId><artifactId>check</artifactId><version>1</version></project>")
        report = run("project", "scan", source, "--facts", fact, "--compact")
        assert any(entry.get("engine") == engine and entry["status"] == "collected" for entry in report["engineFacts"]), report
        print(f"{engine}: installed, upgraded and scanned {item['engine']['version']}")
