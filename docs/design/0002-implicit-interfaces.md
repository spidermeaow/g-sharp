# ADR 0002: Implicit interfaces

**Context:** Composition needs polymorphism without a nominal inheritance graph.

**Decision:** A type satisfies an interface when all public instance methods have
exactly matching names, parameter types, and return types. There is no declaration
of implementation, variance, overload resolution, or automatic numeric conversion.

**Alternatives:** C#-style explicit implementation adds coupling; unrestricted
duck typing sacrifices static guarantees.

**Consequences:** Interface conversion is checked in G-Sharp before Go emission.
Backend method names use a stable shared identity across unrelated types. Private
and static methods never satisfy an interface.
