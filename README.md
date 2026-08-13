# 🧠 Computer Architecture — Processor, Cache & Memory Hierarchy Repository

<div align="center">

![Languages](https://img.shields.io/badge/Languages-Verilog%20HDL%20%7C%20Go-blue?style=for-the-badge)
![Architecture](https://img.shields.io/badge/Focus-MIPS%20%7C%20Cache%20%7C%20Pipelining-brightgreen?style=for-the-badge)
![Simulation](https://img.shields.io/badge/Simulation-Akita%20%7C%20Logisim-purple?style=for-the-badge)
![Course](https://img.shields.io/badge/Course-Computer%20Architecture-orange?style=for-the-badge)
![License](https://img.shields.io/badge/License-MIT-success?style=for-the-badge)

*A comprehensive academic archive of processor design, cache-memory systems, Verilog implementations, architecture simulations, homework assignments, examinations, and course materials for Computer Architecture.*

</div>

---

## 📖 Table of Contents

1. [Overview](#-overview)
2. [Repository Structure](#-repository-structure)
3. [Homework Assignments](#-homework-assignments)
4. [Practical Architecture Designs](#-practical-architecture-designs)
5. [Victim Cache & Memory Hierarchy Project](#-victim-cache--memory-hierarchy-project)
6. [Project Architecture](#-project-architecture)
7. [Benchmarks & Verification](#-benchmarks--verification)
8. [Exams & Course Material](#-exams--course-material)
9. [Toolchain](#-toolchain)
10. [Persian Summary](#-persian-summary-راهنمای-فارسی)
11. [License](#-license)

---

## 🔭 Overview

This repository contains the coursework and practical implementations developed for the **Computer Architecture** course.

The repository progresses from theoretical architecture exercises and digital-system simulations to complete processor implementations in **Verilog HDL**, cache-memory controllers, pipelined processor design, and finally a configurable **multi-level memory hierarchy simulator written in Go and integrated with the Akita simulation framework**.

The main architectural topics represented in the repository include:

* CPU datapath and control
* MIPS-style instruction execution
* Arithmetic Logic Units
* Register files
* Multi-cycle multiplication and division
* Memory interfaces
* Direct-mapped caches
* Multi-level cache hierarchies
* Victim Cache architecture
* FIFO and LRU replacement policies
* Cache hit/miss behavior
* Write-back and eviction behavior
* Processor pipelining
* Memory hierarchy performance analysis
* Average Memory Access Time (AMAT)
* Trace-based architecture simulation
* Application-driven cache benchmarking

---

## 📂 Repository Structure

```text
📦 Computer_Architecture
│
├── 📂 HWs/
│   ├── 📂 1/                     # Theoretical Homework 1
│   ├── 📂 2/                     # Theoretical Homework 2
│   ├── 📂 3/                     # Theoretical Homework 3
│   ├── 📂 4/                     # Theoretical Homework 4
│   ├── 📂 5/                     # Theoretical Homework 5
│   │
│   └── 📂 Practicals/
│       ├── 📂 1/                 # Logisim architecture exercises
│       ├── 📂 2/                 # Logisim architecture exercises
│       ├── 📂 3/                 # Logisim architecture exercises
│       ├── 📂 4/                 # Logisim architecture exercises
│       ├── 📂 5/                 # Verilog processor implementation
│       ├── 📂 6/                 # Cache + processor integration
│       └── 📂 7/                 # Pipelined processor design
│
├── 📂 project/
│   ├── 📄 Computer_Architecture_Projects.pdf
│   ├── 📄 report.pdf
│   │
│   └── 📂 Project/
│       ├── 📂 cmd/
│       │   ├── compare/
│       │   ├── matrixbench/
│       │   ├── mergesortbench/
│       │   ├── sim/
│       │   └── testbench/
│       │
│       ├── 📂 internal/
│       │   ├── benchmark/
│       │   ├── cache/
│       │   ├── config/
│       │   ├── cpu/
│       │   ├── memory/
│       │   ├── metrics/
│       │   ├── model/
│       │   ├── simadapter/
│       │   ├── system/
│       │   └── testbench/
│       │
│       ├── 📄 README.md
│       ├── 📄 AKITA_INTEGRATION.md
│       ├── 📄 APPLICATION_BENCHMARKS.md
│       ├── 📄 TESTBENCH.md
│       ├── 📄 Makefile
│       ├── 📄 go.mod
│       └── 📊 *.csv
│
├── 📂 exams/
│   ├── 📂 midterm/
│   └── 📂 final/
│
├── 📂 slides/
│   ├── 📄 lecture1.pdf
│   ├── 📄 lecture2.pdf
│   ├── 📄 ...
│   └── 📄 lecture 11.pdf
│
├── 📄 LICENSE
└── 📄 README.md
```

---

# 📝 Homework Assignments

## Theoretical Homeworks

The `HWs/1` through `HWs/5` directories contain the written homework assignments completed throughout the semester.

They document the theoretical side of the course and complement the hardware implementations found in the practical assignments.

Solutions and reports are mainly preserved as PDF/ODT documents so that the repository functions both as an implementation portfolio and as an academic course archive.

---

# ⚙️ Practical Architecture Designs

The seven practical assignments show a clear progression from circuit-level architecture design toward processor and memory-system implementation.

## Practical Assignments 1–4 — Digital Architecture in Logisim

The first four practical assignments contain **Logisim (`.circ`) implementations**, assignment specifications, reports, and exported project files.

These exercises establish the low-level hardware concepts required for the later processor implementations.

Each directory generally contains:

```text
Assignment specification
        ↓
Logisim circuit implementation
        ↓
Simulation / verification
        ↓
Written report
```

The original `.circ` files are preserved so the circuits can be inspected and simulated directly using Logisim or Logisim Evolution.

---

## Practical Assignment 5 — 32-bit Processor in Verilog

Practical 5 marks the transition from graphical circuit design to a complete processor implementation using **Verilog HDL**.

The design contains several major processor components:

### Arithmetic Logic Unit

The ALU supports:

```text
00 → ADD
01 → SUB
10 → MUL
11 → DIV
```

Addition and subtraction are combinational, while multiplication and division are implemented as sequential multi-cycle units with `start`, `busy`, and `ready` control behavior.

### Register File

A 32 × 32-bit register file is implemented with:

* Two read ports
* One write port
* Register `$0` permanently mapped to zero
* Synchronous writes
* Reset support

### Control Unit

The control unit decodes MIPS-style opcodes and function fields and generates processor control signals such as:

```text
RegDst
MemtoReg
ALUOp
RegWrite
ALUSrc
MemRead
MemWrite
Branch
Jump
Jal
Jr
```

Supported instruction behavior includes operations such as:

```text
ADD
SUB
MUL
DIV
LW
SW
ADDI
SUBI
LUI
BEQ
J
JAL
JR
```

### Processor Datapath

The processor integrates:

```text
              ┌──────────────┐
Instruction → │ Control Unit │
              └──────┬───────┘
                     │
                     v
PC → Instruction → Register File → ALU → Memory
                     ^                    │
                     └──── Write Back ────┘
```

The design therefore represents a functional **32-bit MIPS-inspired processor datapath** rather than an isolated collection of Verilog modules.

---

## Practical Assignment 6 — Cache Memory & Processor Integration

Practical 6 introduces the **memory hierarchy**.

### Question 1 — Direct-Mapped L1 Cache

A hardware cache controller is implemented in Verilog between the CPU and main memory.

The cache contains:

* 64 cache entries
* 128-bit / 16-byte cache blocks
* Address decomposition into:

  * Tag
  * Index
  * Block offset
* Valid bits
* Cache-hit detection
* Block allocation
* Read miss handling
* Write handling
* CPU/Main-Memory handshake signals

The cache controller is implemented as an FSM with states similar to:

```text
IDLE
 ↓
READ_REQ → READ_WAIT
 ↓
ALLOC_REQ → ALLOC_WAIT
 ↓
WRITE_REQ → WRITE_WAIT
```

This implementation demonstrates how an architectural cache model translates into actual control logic.

### Question 2 — Processor + Cache System

The second part combines the processor architecture from the previous assignment with cache memory.

Instead of communicating directly with memory, processor instruction/data requests pass through cache controllers.

Conceptually:

```text
                    ┌───────────────┐
                    │     CPU       │
                    └───────┬───────┘
                            │
                 ┌──────────┴──────────┐
                 │                     │
                 v                     v
        Instruction Cache        Data Cache
                 │                     │
                 └──────────┬──────────┘
                            v
                       Main Memory
```

This assignment connects processor design with realistic memory-access behavior and prepares the architecture for the final cache-hierarchy project.

---

## Practical Assignment 7 — Pipelined Processor

Practical 7 evolves the processor implementation into a **pipelined architecture**.

Explicit pipeline registers are introduced between the major processing stages:

```text
Fetch
  │
  ▼
F/D Register
  │
  ▼
Decode
  │
  ▼
D/E Register
  │
  ▼
Execute
  │
  ▼
E/M Register
  │
  ▼
Memory
  │
  ▼
M/W Register
  │
  ▼
Write Back
```

The implementation maintains processor state across pipeline stages using dedicated stage registers and propagates instruction data, control information, register operands, ALU outputs, and write-back information through the pipeline.

This assignment demonstrates the architectural transition from sequential instruction execution toward **instruction-level parallelism**.

---

# 🚀 Victim Cache & Memory Hierarchy Project

The most substantial implementation in this repository is located in:

```text
project/Project/
```

The project is a **configurable memory-hierarchy simulator written in Go** and integrated with **Akita v4.9.0**.

Its primary objective is to investigate the effect of a **Victim Cache** on cache conflicts, latency, hit rate, total execution cycles, and memory-hierarchy behavior.

---

## 🧱 Supported Memory Hierarchies

The simulator supports four configurations:

```text
memory
CPU → Main Memory
```

```text
l1
CPU → L1 → Main Memory
```

```text
l1-l2
CPU → L1 → L2 → Main Memory
```

```text
full
CPU → L1 → Victim Cache → L2 → Main Memory
```

This makes it possible to compare progressively more advanced memory hierarchies using identical workloads.

---

## 🎯 Victim Cache

The Victim Cache is a small, fully associative cache located between L1 and L2.

Its purpose is to retain blocks recently evicted from L1 so that conflict misses can be resolved without accessing the slower lower levels of the hierarchy.

```text
CPU
 │
 ▼
L1 Cache
 │
 ├── Hit ───────────────► CPU
 │
 └── Miss
      │
      ▼
 Victim Cache
      │
      ├── Hit → Swap / Restore block
      │
      └── Miss
           │
           ▼
         L2 Cache
           │
           ▼
       Main Memory
```

Two replacement policies are implemented for the Victim Cache:

* **FIFO — First In, First Out**
* **LRU — Least Recently Used**

This allows the simulator to measure how replacement policy influences cache behavior.

---

# 🏗️ Project Architecture

The Go project is separated into independent packages so that cache behavior, simulation infrastructure, benchmarking, metrics, and command-line tools remain modular.

```text
Project/
│
├── cmd/
│   ├── sim/             # Run a single architecture/workload
│   ├── compare/         # Compare configurations
│   ├── testbench/       # Complete automated validation suite
│   ├── matrixbench/     # Matrix multiplication benchmark
│   └── mergesortbench/  # Merge-sort benchmark
│
└── internal/
    ├── benchmark/       # Synthetic & application workloads
    ├── cache/           # L1, L2 and Victim Cache implementations
    ├── config/          # Architecture configuration
    ├── cpu/             # Memory-request generation
    ├── memory/          # Main-memory model
    ├── metrics/         # Statistics & AMAT
    ├── model/           # Request/response/block models
    ├── simadapter/      # Akita integration layer
    ├── system/          # Complete memory hierarchy
    └── testbench/       # Automated validation infrastructure
```

---

# 🔌 Akita Simulation Integration

The simulator uses **Akita v4.9.0** as its event-driven simulation infrastructure.

The execution path is:

```text
Benchmark Requests
        │
        ▼
MemoryRequestDriver
   (Akita Component)
        │
        ▼
 accessRequestMsg
        │
        ▼
DirectConnection
        │
        ▼
MemoryHierarchyExecutor
   (Akita Component)
        │
        ▼
System.Access
        │
        ▼
L1 → Victim → L2 → Memory
        │
        ▼
Scheduled Completion Event
        │
        ▼
 accessResponseMsg
        │
        ▼
MemoryRequestDriver
```

Akita is responsible for:

* Simulation engine execution
* Components
* Ports
* Typed messages
* Direct connections
* Scheduled events
* Request/response delivery
* Simulated timing

The functional cache hierarchy remains encapsulated in the project's `System` implementation, allowing the architecture logic and simulation infrastructure to remain cleanly separated.

---

# 📊 Benchmarks & Verification

The simulator includes both **synthetic memory traces** and **real algorithmic workloads**.

## Synthetic Workloads

### `repeated`

Repeated accesses to the same memory region demonstrate L1 warm-up and cache hits.

### `sequential`

Sequential word accesses demonstrate **spatial locality** and cache-block utilization.

### `conflict`

Addresses are intentionally selected to collide in the direct-mapped L1 cache.

This workload highlights the principal benefit of the Victim Cache.

### `mixed`

A larger deterministic trace exercises all hierarchy levels and creates meaningful differences between FIFO and LRU Victim Cache policies.

---

## Application Benchmarks

Two real algorithms are also instrumented as memory workloads.

### Matrix Multiplication

The simulator records the logical memory accesses produced while multiplying square matrices.

The benchmark compares:

```text
L1 + L2
L1 + Victim(FIFO) + L2
L1 + Victim(LRU) + L2
```

### Merge Sort

A top-down merge-sort workload records accesses to both the primary array and temporary scratch memory.

It is used to evaluate cache behavior on a workload with a memory-access pattern different from matrix multiplication.

---

# 📈 Performance Metrics

The simulator collects statistics including:

* L1 accesses, hits, and misses
* Victim Cache hits and misses
* L2 hits and misses
* Main-memory accesses
* Dirty write-backs
* Total cycles
* Per-level latency
* Hit rates
* Average Memory Access Time (AMAT)

Results can be exported to CSV for analysis and report generation.

---

# 🧪 Running the Memory Hierarchy Simulator

Enter the project directory:

```bash
cd project/Project
```

Install Go dependencies:

```bash
go mod download
```

Run a simple simulation:

```bash
go run ./cmd/sim
```

Run a specific topology:

```bash
go run ./cmd/sim -topology l1-l2 -trace conflict
```

Run the complete Victim Cache hierarchy:

```bash
go run ./cmd/sim \
    -topology full \
    -trace mixed \
    -victim=true \
    -victim-policy=LRU
```

---

## Compare Architectures

```bash
go run ./cmd/compare -trace conflict
```

Compare both Victim Cache policies:

```bash
go run ./cmd/compare \
    -trace all \
    -victim-policy BOTH
```

---

## Complete Automated Testbench

```bash
go run ./cmd/testbench
```

Export results:

```bash
go run ./cmd/testbench -csv results.csv
```

Run one workload:

```bash
go run ./cmd/testbench -trace mixed
```

---

## Application Benchmarks

Matrix multiplication:

```bash
go run ./cmd/matrixbench
```

Custom matrix size:

```bash
go run ./cmd/matrixbench \
    -size 12 \
    -csv matrix-results.csv
```

Merge sort:

```bash
go run ./cmd/mergesortbench
```

Custom input length:

```bash
go run ./cmd/mergesortbench \
    -length 32 \
    -csv mergesort-results.csv
```

---

# ✅ Verification

The Go implementation includes unit tests and integration tests for the major architecture components.

Run all tests:

```bash
go test ./...
```

Run the Go race detector:

```bash
go test -race ./...
```

Run static analysis:

```bash
go vet ./...
```

The verification infrastructure checks both individual cache components and the behavior of the complete hierarchy across multiple workloads and configurations.

---

# 🛠️ Toolchain

The repository uses several tools across different stages of the course.

### Verilog HDL

Used for processor, cache, and pipelined architecture implementation.

Useful open-source tools:

```bash
sudo apt install iverilog gtkwave
```

Compile a Verilog design:

```bash
iverilog -o design.out design.v
```

Run:

```bash
vvp design.out
```

---

### Logisim / Logisim Evolution

Used for the earlier graphical digital-architecture assignments stored as `.circ` files.

These files allow direct inspection of gates, datapaths, registers, multiplexers, memories, and other architecture components.

---

### Go

The final memory-hierarchy simulator is implemented in Go.

Check the installation:

```bash
go version
```

The project uses Go modules for dependency management.

---

### Akita

The final project uses the Akita simulation framework to provide event-driven architecture simulation.

The project is pinned to:

```text
github.com/sarchlab/akita/v4 v4.9.0
```

---

# 📝 Exams & Course Material

## Exams

The `exams/` directory contains both course examinations and completed solutions.

```text
exams/
├── midterm/
│   ├── میانترم.pdf
│   └── CA_midterm_403106681.pdf
│
└── final/
    ├── final.pdf
    └── CA_Final_403106681.pdf
```

This preserves both the original exam material and the corresponding completed work.

---

## Lecture Slides

The `slides/` directory contains the lecture material used throughout the semester.

```text
lecture1.pdf
lecture2.pdf
lecture3.pdf
lecture 4.pdf
lecture 5.pdf
lecture 6.pdf
lecture 7.pdf
lecture 8.pdf
lecture 9.pdf
lecture 10.pdf
lecture 11.pdf
```

Together with the homework and implementations, these slides make the repository a complete archive of the course.

---

# 🇮🇷 Persian Summary — راهنمای فارسی

<details>

<summary><strong>برای مشاهده توضیحات فارسی کلیک کنید</strong></summary>

<br>

## درباره این مخزن

این ریپازیتوری آرشیو درس **معماری کامپیوتر** است و از تمرین‌های تئوری و طراحی مدارهای ساده‌تر شروع می‌شود و تا پیاده‌سازی پردازنده، حافظهٔ Cache، پردازندهٔ Pipeline و پروژهٔ نهایی سلسله‌مراتب حافظه ادامه پیدا می‌کند.

ساختار کلی مسیر عملی درس تقریباً به شکل زیر است:

```text
طراحی مدار در Logisim
        ↓
پردازنده با Verilog
        ↓
طراحی Cache
        ↓
اتصال Cache به پردازنده
        ↓
طراحی پردازنده Pipeline
        ↓
شبیه‌سازی کامل Memory Hierarchy
```

### تمرین‌های عملی ۱ تا ۴

شامل فایل‌های مدار Logisim، گزارش‌ها و صورت تمرین‌های مربوط به مباحث اولیهٔ معماری و طراحی سخت‌افزار هستند.

### تمرین عملی ۵

یک پردازندهٔ ۳۲ بیتی MIPS-like با Verilog پیاده‌سازی شده است.

بخش‌های مهم آن شامل:

* ALU
* Register File
* Control Unit
* Program Counter
* مسیر Write Back
* دستورات Load/Store
* Branch و Jump
* ضرب و تقسیم چندکلاکی

است.

### تمرین عملی ۶

تمرکز این تمرین روی **Cache Memory** است.

در بخش اول یک L1 Cache مستقیم‌نگاشت پیاده‌سازی شده و در بخش بعدی Cache به پردازنده متصل شده است تا درخواست‌های Instruction و Data به‌جای دسترسی مستقیم به حافظه از Cache عبور کنند.

### تمرین عملی ۷

پردازندهٔ قبلی به معماری **Pipeline** گسترش داده شده و رجیسترهای بین مراحل مختلف پردازنده اضافه شده‌اند.

مسیر کلی به صورت:

```text
Fetch → Decode → Execute → Memory → Write Back
```

است.

---

## پروژهٔ اصلی — Victim Cache

مهم‌ترین بخش ریپو پروژهٔ داخل مسیر زیر است:

```text
project/Project
```

در این پروژه یک شبیه‌ساز سلسله‌مراتب حافظه با زبان **Go** ساخته شده است.

چهار معماری مختلف قابل شبیه‌سازی هستند:

```text
CPU → Memory

CPU → L1 → Memory

CPU → L1 → L2 → Memory

CPU → L1 → Victim Cache → L2 → Memory
```

هدف اصلی بررسی تأثیر **Victim Cache** روی Conflict Missهای L1 است.

Victim Cache دو سیاست جایگزینی دارد:

```text
FIFO
LRU
```

و می‌توان عملکرد آن‌ها را با یکدیگر مقایسه کرد.

---

## Akita

برای بخش شبیه‌سازی event-driven پروژه از فریم‌ورک **Akita** استفاده شده است.

Akita مدیریت مواردی مثل:

* Engine
* Component
* Port
* Message
* Connection
* Event
* Simulation Time

را انجام می‌دهد.

در عین حال منطق اصلی L1، Victim Cache، L2 و Main Memory در بخش functional پروژه نگه داشته شده تا شبیه‌سازی و منطق معماری از هم جدا باقی بمانند.

---

## Benchmarkها

علاوه بر Traceهای مصنوعی مثل:

```text
repeated
sequential
conflict
mixed
```

دو الگوریتم واقعی نیز برای تولید Memory Trace استفاده شده‌اند:

```text
Matrix Multiplication
Merge Sort
```

به این ترتیب می‌توان بررسی کرد اضافه‌شدن Victim Cache و تغییر سیاست FIFO به LRU چه اثری روی تعداد Hit/Missها و Cycleهای کل برنامه دارد.

---

## نتیجه

این مخزن در مجموع بخش مهمی از مسیر درس معماری کامپیوتر را پوشش می‌دهد:

```text
CPU Datapath
MIPS Architecture
ALU
Register File
Control Unit
Memory System
Cache
Victim Cache
L1 / L2
FIFO / LRU
Pipelining
Performance Metrics
AMAT
Event-driven Simulation
```

و پروژهٔ نهایی آن یک نمونهٔ عملی از تبدیل مفاهیم تئوری Cache و سلسله‌مراتب حافظه به یک شبیه‌ساز معماری قابل تست و Benchmark است.

</details>

---

# 📄 License

This repository is open-source and released under the [MIT License](LICENSE).

The material is preserved as an academic and engineering reference for students interested in **computer architecture, processor design, Verilog HDL, cache systems, and memory hierarchy simulation**.

<div align="center">

<sub>Developed and maintained by <b>M. Mahdi Moradi</b> (@mahdi0x06).</sub>

</div>
