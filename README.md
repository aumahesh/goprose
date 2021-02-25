# goprose
ProSe compiler in golang (generates golang code)

[![Go](https://github.com/aumahesh/goprose/actions/workflows/go.yml/badge.svg)](https://github.com/aumahesh/goprose/actions/workflows/go.yml)

## ProSe: Overview

ProSe is a programming tool that allows the designers to concisely specify
programs (for example, network protocols). ProSe is based on the theoretical
foundation on computational model in sensor networks. ProSe enables the
designer to specify sensor network protocols and macroprogramming
primitives in simple, abstract models considered in distributed computing
literature (for example, shared-memory, message-passing, etc). Furthermore,
ProSe enables the reuse of existing fault-tolerance/self-stabilizing algorithms
from the literature in the context of sensor networks. ProSe automatically
generate and deploy code. An advantage of ProSe is that it will facilitate
the designer to use existing algorithms for automating the addition of
fault-tolerance to existing programs. Moreover, since abstract models are
used to specify protocols, ProSe allows the designer to gain assurance about
the programs deployed in the network using tools such as model checkers.

ProSe programs are written in guarded commands (cf. [1], [2])

### Guarded Commands

A program is a set of variables and a finite set of actions. Each variable
has a predefined nonempty domain. In this dissertation, the programs are
specified in terms of guarded commands each guarded command is of
the form: 
    
```
    guard   ->  statement;
```

The guard of each action is a predicate over the program variables. The
statement of each action atomically updates zero or more program variables.

## Example ProSe Program

Here is an example program written in guarded commands. This is a routing
tree maintenance program (cf. [3]). The program is available [here](https://github.com/aumahesh/goprose/blob/main/proseFiles/routing.prose).

```
program RoutingTreeMaintenance
import "neighborhood"
import "math/rand"
sensor j
const
	int CMAX = 100;
var
	public int inv.j, dist.j = 0, rand.Int63n(10);
	private string p.j;
begin
	(dist.k < dist.j) && (neighborhood.up(k)) && (inv.k < CMAX) && (inv.k < inv.j)
		-> p.j, inv.j = k, inv.k;
	|
	(dist.k < dist.j) && (neighborhood.up(k)) && (inv.k+1 < CMAX) && (inv.k+1 < inv.j)
		-> p.j, inv.j = k, inv.k+1;
	|
	(p.j != "") && (neighborhood.up(p.j) == false)|| (inv.(p.j) >= CMAX) ||
		( (dist.(p.j) < dist.j) && (inv.j != inv.(p.j)) ) ||
		( (dist.(p.j) > dist.j) && (inv.j != inv.(p.j)+1) )
		-> p.j, inv.j = "", CMAX;
	|
	(p.j == "") && (inv.j < CMAX)
		-> inv.j = CMAX;
end

```

The complete grammar for ProSe programs is available [here](https://github.com/aumahesh/goprose/blob/main/internal/parser/prose.go#L1). 

## Generated Code

goprose compiles the program into Go Module. 

```
make all
bin/prose --p proseFiles/routing.prose --o _examples/
```

Generated module is written to `_examples` folder. Generated code is available [here](https://github.com/aumahesh/goprose/tree/main/_examples/RoutingTreeMaintenance)

```
_examples/RoutingTreeMaintenance
├── Makefile
├── cmd
│   └── main.go
├── go.mod
├── go.sum
├── internal
│   ├── RoutingTreeMaintenance_impl.go
│   └── RoutingTreeMaintenance_intf.go
└── proto
    └── state.proto

3 directories, 7 files
```

## Publications

1. R. Hajisheykhi, L. Zhu, M. Arumugam, M. Demirbas, and S. Kulkarni.  “Slow is Fast” for Wireless Sensor Networks in the Presence of Message Losses. Journal of Parallel and Distributed Computing (JPDC), Volume 77,Pages 41–57, 03/15.
2. M. Arumugam, M. Demirbas, and S. Kulkarni. “Slow is Fast” for Wireless Sensor Networks in the Presence of Message Losses. In Proceedings of the 12th International Symposium on Stabilization, Safety, and Security of Distributed Systems (SSS), 09/10. (New York City, NY).
3. M. Arumugam and S. S. Kulkarni. ProSe: A Programming Tool for Rapid Prototyping of Sensor Networks. In Proceedings of the First International Conference on Sensor Systems and Software (S-Cube), 09/09. (Pisa, Italy).
4. M. Arumugam, L. Wang, and S. S. Kulkarni. A Case Study on Prototyping Power Management Protocols for Sensor Networks. In Proceedings of the Eighth International Symposium on Stabilization, Safety, and Security of Distributed Systems, 11/06. (Dallas, TX).
5. M. Arumugam. Rapid Prototyping and Quick Deployment of Sensor Networks. Ph.D. Dissertation, 2006.
6. https://aumahesh.me/

## References

1. Parallel Program Design: A Foundation. K. M. Chandy and J. Misra, 1988
2. A Discipline of Programming, E. W. Dijkstra, 1997.
3. Stabilization of grid routing in sensor networks. Y-R. Choi, M. G. Gouda, H. Zhang, and A. Arora.
   AIAA Journal of Aerospace Computing, Information, and Communication (JACIC), 2006.
