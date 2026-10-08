package com.edwardnoaland.scryer.analyze

import com.sun.source.tree.*
import com.sun.source.util.TreePath
import com.sun.source.util.TreeScanner
import com.sun.source.util.Trees
import javax.lang.model.element.Element
import javax.lang.model.element.ExecutableElement
import javax.lang.model.element.TypeElement
import javax.lang.model.type.TypeMirror

/** Bounded local constant propagation; target code is never executed. */
internal class LiteralReflection(
    private val trees: Trees,
    private val descriptor: (TypeMirror) -> String?,
    private val lookup: (String, String, List<String>, Boolean) -> SymbolId?,
) {
    private sealed interface Value
    private data class ClassValue(val owner: String) : Value
    private data class MethodValue(val id: SymbolId) : Value

    fun resolve(unit: CompilationUnitTree): Map<MethodInvocationTree, SymbolId> {
        val resolved = mutableMapOf<MethodInvocationTree, SymbolId>()
        fun path(node: Tree) = TreePath.getPath(unit, node)
        fun element(node: Tree) = trees.getElement(path(node))
        object : TreeScanner<Unit, Unit>() {
            override fun visitMethod(node: MethodTree, unused: Unit?) {
                val values = mutableMapOf<Element, Value>()
                fun evaluate(expression: ExpressionTree): Value? {
                    if (expression is IdentifierTree) return element(expression)?.let { values[it] }
                    if (expression is ParenthesizedTree) return evaluate(expression.expression)
                    if (expression !is MethodInvocationTree) return null
                    val method = element(expression) as? ExecutableElement ?: return null
                    val owner = (method.enclosingElement as? TypeElement)?.qualifiedName?.toString()
                    val name = method.simpleName.toString()
                    val select = expression.methodSelect as? MemberSelectTree
                    val receiver = select?.expression?.let(::evaluate)
                    if (owner == "java.lang.Class" && name == "forName" && expression.arguments.size == 1) {
                        return ((expression.arguments.single() as? LiteralTree)?.value as? String)?.let(::ClassValue)
                    }
                    if (owner == "java.lang.Class" && name in listOf("getMethod", "getDeclaredMethod") && receiver is ClassValue) {
                        val methodName = (expression.arguments.firstOrNull() as? LiteralTree)?.value as? String ?: return null
                        val parameters = expression.arguments.drop(1).map { argument ->
                            val literal = argument as? MemberSelectTree ?: return null
                            if (literal.identifier.toString() != "class") return null
                            descriptor(trees.getTypeMirror(path(literal.expression))) ?: return null
                        }
                        return lookup(receiver.owner, methodName, parameters, name == "getDeclaredMethod")?.let(::MethodValue)
                    }
                    if (owner == "java.lang.reflect.Method" && name == "invoke" && receiver is MethodValue) {
                        resolved[expression] = receiver.id
                    }
                    return null
                }
                fun inspect(expression: ExpressionTree) {
                    var unsafe = false
                    object : TreeScanner<Unit, Unit>() {
                        override fun visitAssignment(node: AssignmentTree, unused: Unit?) { unsafe = true }
                        override fun visitCompoundAssignment(node: CompoundAssignmentTree, unused: Unit?) { unsafe = true }
                        override fun visitLambdaExpression(node: LambdaExpressionTree, unused: Unit?) { unsafe = true }
                    }.scan(expression, Unit)
                    if (unsafe) { values.clear(); return }
                    object : TreeScanner<Unit, Unit>() {
                        override fun visitMethodInvocation(node: MethodInvocationTree, unused: Unit?) {
                            evaluate(node)
                            super.visitMethodInvocation(node, unused)
                        }
                    }.scan(expression, Unit)
                }
                fun block(body: BlockTree) {
                    for (statement in body.statements) {
                        when (statement) {
                            is VariableTree -> {
                                statement.initializer?.let(::inspect)
                                val key = element(statement)
                                val value = statement.initializer?.let(::evaluate)
                                if (key != null && value != null) values[key] = value
                            }
                            is ExpressionStatementTree -> inspect(statement.expression)
                            is ReturnTree -> statement.expression?.let(::inspect)
                            is BlockTree -> { block(statement); values.clear() }
                            is TryTree -> {
                                if (statement.resources.isEmpty()) block(statement.block)
                                values.clear()
                            }
                            else -> values.clear()
                        }
                    }
                }
                node.body?.let(::block)
                super.visitMethod(node, unused)
            }
        }.scan(unit, Unit)
        return resolved
    }
}
