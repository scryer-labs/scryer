package com.edwardnoaland.scryer.cli

import java.io.PrintStream

internal val CLI_USAGE = "$SCAN_USAGE\n$ANALYZE_USAGE"

/** Handles global help and dispatch; each command owns its arguments and execution. */
fun runCli(args: Array<String>, out: PrintStream, err: PrintStream): Int {
    if (args.contentEquals(arrayOf("--help")) || args.contentEquals(arrayOf("-h"))) {
        out.println(CLI_USAGE)
        return 0
    }

    val commandArgs = args.drop(1).toTypedArray()
    return when (args.firstOrNull()) {
        "scan" -> runScanCommand(commandArgs, out, err)
        "analyze" -> runAnalyzeCommand(commandArgs, out, err)
        else -> {
            err.println(CLI_USAGE)
            2
        }
    }
}
