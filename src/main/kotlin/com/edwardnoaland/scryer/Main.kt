package com.edwardnoaland.scryer

import com.edwardnoaland.scryer.cli.runCli
import kotlin.system.exitProcess

fun main(args: Array<String>) {
    exitProcess(runCli(args, System.out, System.err))
}
