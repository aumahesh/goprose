# goprose

[![Go](https://github.com/aumahesh/goprose/actions/workflows/go.yml/badge.svg)](https://github.com/aumahesh/goprose/actions/workflows/go.yml)

A Go compiler for **ProSe** — a language for specifying distributed sensor-network algorithms as guarded commands. Write a `.prose` file, run the compiler, get a complete, deployable Go module.

## What is ProSe?

Distributed algorithms are often described in the literature as sets of *guarded commands* ([Chandy & Misra, 1988](https://www.amazon.com/Parallel-Program-Design-Mani-Chandy/dp/0201058669)):

```
guard → statement
```

ProSe lets you write programs in exactly that style. The compiler handles all the networking boilerplate (UDP multicast, protobuf state broadcast, neighbor tracking) so you can focus on the algorithm.

Each round, all guards are evaluated. One of the commands whose guard is true is picked at random and executed. This matches the shared-memory computational model used in distributed computing theory. See [Arumugam & Kulkarni, S-Cube 2009](paper/Arumugam-Sensor%20Systems%20and%20Software-2010-First%20International%20Conference%20on%20Sensor%20Systems%20and%20Software%20S-Cube.pdf) for the original ProSe paper.

## Quick start

```bash
make build                                    # produces bin/prose
bin/prose -p proseFiles/gcd.prose -o _examples/
```

The compiler writes a self-contained Go module to `_examples/<ProgramName>/`. Build and run it with the generated `Makefile`.

## Writing a ProSe program

### Structure

```
program <Name>

[import "<go-package>"]

sensor <id>       # name for "this node"

[const
    <type> <Name> = <value>;]

var
    <access> <type> <name>.<sensor> [= <value>];

begin
    <statement>
  | <statement>
  ...
end
```

`<access>` is `public` (broadcast to neighbors) or `private` (local only). Variables are referenced as `x.j` (local) or `x.k` (neighbor).

### Guarded commands

The classic form — a boolean guard followed by one or more assignments:

```
program GCD
sensor j
var
    public int X.j, Y.j = 3542, 943;
begin
    X.j > Y.j -> X.j = X.j - Y.j;
  |
    Y.j > X.j -> Y.j = Y.j - X.j;
end
```

Multiple assignments in one action are comma-separated on both sides:

```
    (dist.k < dist.j) -> p.j, dist.j = k, dist.k;
```

### Priority

Prefix a statement with `<N>` to fire it once every N rounds instead of every round. Higher N = less frequent.

```
begin
    <2> isEvaderHere.j -> p.j = j; dist.j = 0;
  |
    <1> dist.k < dist.j -> p.j = k; dist.j = dist.k + 1;
end
```

### Dijkstra's Guarded Command Language (GCL)

Two additional statement forms from Dijkstra's GCL [[2]](#references) are supported at the top level.

**`if…fi` — alternative construct** ([Dijkstra, 1997](https://www.amazon.com/Discipline-Programming-Edsger-W-Dijkstra/dp/013215871X)). Pick one true-guarded command non-deterministically. No-op if none are true.

```
program dijkstra_gcl
sensor j
var
    public int x.j = 42;
    public int y.j = 18;
begin
    if x.j > y.j -> x.j = x.j - y.j;
    | y.j > x.j -> y.j = y.j - x.j;
    fi
end
```

**`do…od` — repetitive construct** ([Dijkstra, 1997](https://www.amazon.com/Discipline-Programming-Edsger-W-Dijkstra/dp/013215871X)). Loop, picking a true-guarded command each iteration, until none are true.

```
program dijkstra_gcl_do
sensor j
var
    public int x.j = 10;
    public int y.j = 3;
begin
    do x.j > y.j -> x.j = x.j - y.j;
    | x.j > 0 && x.j <= y.j -> x.j = x.j - 1;
    od
end
```

Both forms can be mixed with regular guarded statements in the same `begin…end` block.

## What gets generated

The compiler writes a complete Go module:

```
_examples/GCD/
├── Makefile
├── cmd/main.go
├── go.mod
├── internal/
│   ├── ProSe_impl_GCD.go
│   └── ProSe_intf_GCD.go
└── proto/state.proto
```

Each statement in your `.prose` file becomes a guard/action function pair. For the GCD example:

```go
// Guard: pure predicate, no side effects
func (this *ProSe_impl_GCD) evaluateGuard0() (bool, *NeighborState) {
    ...
    if (this.state.X > this.state.Y) {
        takeAction = true
    }
    return takeAction, neighbor
}

// Action: executes when this guard fires
func (this *ProSe_impl_GCD) executeAction0(neighbor *NeighborState) (bool, *NeighborState) {
    ...
    this.state.X = (this.state.X - this.state.Y)
    ...
}
```

The scheduler evaluates all guards each round and picks one winner at random:

```go
func (this *ProSe_impl_GCD) updateLocalState() bool {
    couldExecute := []int{}
    for index, stmtFunc := range this.guards {
        if ok, nbr := stmtFunc(); ok {
            couldExecute = append(couldExecute, index)
        }
    }
    if len(couldExecute) > 0 {
        actionIndex := rand.Intn(len(couldExecute))
        this.actions[couldExecute[actionIndex]](...)
        return true
    }
    return false
}
```

For a GCL `if…fi` statement, the compiler generates the non-deterministic selection inline in the action:

```go
func (this *ProSe_impl_dijkstra_gcl) executeAction0(neighbor *NeighborState) (bool, *NeighborState) {
    ...
    var temp0 []int
    if (this.state.X > this.state.Y) { temp0 = append(temp0, 0) }
    if (this.state.Y > this.state.X) { temp0 = append(temp0, 1) }
    if len(temp0) > 0 {
        switch temp0[rand.Intn(len(temp0))] {
        case 0: this.state.X = (this.state.X - this.state.Y)
        case 1: this.state.Y = (this.state.Y - this.state.X)
        }
    }
    ...
}
```

And for `do…od`, a `for` loop that breaks when no guard is true:

```go
    for {
        var temp0 []int
        if (this.state.X > this.state.Y) { temp0 = append(temp0, 0) }
        if (this.state.X > 0 && this.state.X <= this.state.Y) { temp0 = append(temp0, 1) }
        if len(temp0) == 0 { break }
        switch temp0[rand.Intn(len(temp0))] {
        case 0: this.state.X = (this.state.X - this.state.Y)
        case 1: this.state.X = (this.state.X - int64(1))
        }
    }
```

## More examples

| File | Algorithm |
|------|-----------|
| `proseFiles/gcd.prose` | GCD via repeated subtraction |
| `proseFiles/max.prose` | Distributed max |
| `proseFiles/routing.prose` | Routing tree maintenance |
| `proseFiles/distributed_reset.prose` | Distributed reset |
| `proseFiles/pursuer_evader_with_priority.prose` | Pursuer-evader tracking with priority |
| `proseFiles/dijkstra_gcl.prose` | GCD via GCL `if…fi` |
| `proseFiles/dijkstra_gcl_do.prose` | Modulo via GCL `do…od` |

Full grammar: [`internal/parser/prose.go`](internal/parser/prose.go)

