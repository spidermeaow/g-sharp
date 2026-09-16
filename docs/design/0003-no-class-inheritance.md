# ADR 0003: Composition, copies, and references

**Context:** The C# object model is much larger than this language needs.

**Decision:** No inheritance. Classes have reference semantics, structs have value
semantics, and methods on structs receive a copy. Auto-properties are checked
storage members; access checks happen in the frontend. No getters with user code.
Null syntax is unavailable, and implicit unsafe reference field initialization is
rejected. Field initializers are constant expressions in the first bootstrap.

**Alternatives:** Pointer receivers for structs would make interface/value copying
surprising. Silent null zero values would introduce an undocumented nullable model.

**Consequences:** Struct mutation methods do not mutate callers. Nested references
still share their objects. Rich composition through initialized reference fields
needs constructors or explicit optional values in a later milestone.
