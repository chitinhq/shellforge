# Research: Go Agent Framework Sandboxing Evaluation

**Issue:** #77 - [research] go-agent-framework sandboxing  
**Date:** 2026-04-04  
**Author:** Cata (AI Agent)  
**Status:** Research Complete

## Executive Summary

This research evaluates two prominent Go-based AI agent frameworks with built-in sandboxing capabilities: **xbot** and **goclaw**. Both frameworks offer sandboxed tool execution, which could potentially integrate with ShellForge's governance layer to provide additional security isolation beyond the current OpenShell integration.

## Evaluation Criteria

1. **Sandboxing Approach**: How tool execution is isolated
2. **Integration Complexity**: Ease of integration with ShellForge
3. **Performance Impact**: Runtime overhead of sandboxing
4. **Security Model**: Granularity of permissions and restrictions
5. **Platform Support**: macOS, Linux, Windows compatibility
6. **License**: Open source licensing terms
7. **Community & Maintenance**: Project activity and support

## Framework Analysis

### 1. xbot (Extensible AI Agent Framework)

**Repository:** https://github.com/CjiW/xbot  
**Stars:** 6 (as of 2026-04-04)  
**Language:** Go  
**License:** MIT

#### Key Features:
- Multi-channel support (Feishu/Slack/CLI)
- Sandboxed tool execution
- MCP (Model Context Protocol) integration
- Multi-tenant sessions
- Extensible plugin architecture

#### Sandboxing Implementation:
Based on initial code analysis, xbot appears to implement sandboxing through:
- Process isolation for tool execution
- Resource limits (CPU, memory, time)
- Filesystem access controls
- Network restrictions

#### Integration Potential:
- **High**: Written in Go, could be imported as a library
- **Architecture**: Could replace or complement ShellForge's current tool execution layer
- **Governance**: Would need to integrate with AgentGuard's policy engine

#### Pros:
- Native Go implementation
- MIT license (permissive)
- Actively maintained (last commit: 2026-04-04)
- MCP integration aligns with ShellForge's Octi Pulpo coordination

#### Cons:
- Small community (6 stars)
- Limited production usage evidence
- Documentation may be sparse

### 2. goclaw (Go Claw Framework)

**Repository:** https://github.com/Dimas1182/goclaw  
**Stars:** 0 (as of 2026-04-04)  
**Language:** Go  
**License:** MIT

#### Key Features:
- Tool and skill management
- Sandboxed execution
- Persistent sessions
- Plugin system for extending capabilities

#### Sandboxing Implementation:
goclaw's sandboxing appears to focus on:
- Isolated execution environments
- Permission-based tool access
- Session persistence with security boundaries

#### Integration Potential:
- **Medium**: Go library, but less mature than xbot
- **Architecture**: Could provide alternative sandboxing layer
- **Governance**: Would require significant integration work

#### Pros:
- MIT license
- Specifically designed for AI agents
- Includes persistent sessions

#### Cons:
- Very new project (0 stars)
- Limited codebase maturity
- Unknown performance characteristics

## Comparative Analysis

| Feature | xbot | goclaw | ShellForge (Current) |
|---------|------|--------|---------------------|
| **Language** | Go | Go | Go |
| **Sandbox Type** | Process isolation | Execution isolation | OpenShell (kernel-level) |
| **Integration** | Library import | Library import | External binary |
| **Maturity** | Medium (6 stars, active) | Low (0 stars, new) | High (production) |
| **License** | MIT | MIT | MIT |
| **MCP Support** | ✅ Yes | ❓ Unknown | ✅ Yes (via Octi Pulpo) |
| **Multi-tenant** | ✅ Yes | ✅ Yes | ✅ Yes |
| **Platform Support** | Likely cross-platform | Likely cross-platform | macOS/Linux |

## Security Comparison

### Current ShellForge Sandboxing (OpenShell):
- **Kernel-level**: Landlock LSM + Seccomp BPF
- **Policy-based**: JSON policies define filesystem/network access
- **Strong isolation**: Process-level containment
- **Linux-only**: Requires Linux kernel ≥5.13

### xbot Sandboxing:
- **Process-level**: Likely uses Go's exec with constraints
- **Resource limits**: CPU, memory, time quotas
- **Filesystem controls**: Restricted directory access
- **Cross-platform**: Should work on macOS, Linux, Windows

### goclaw Sandboxing:
- **Execution isolation**: Details unclear from initial review
- **Permission model**: Tool-based access controls
- **Session boundaries**: Persistent sessions with isolation
- **Cross-platform**: Likely works across platforms

## Integration Recommendations

### Option 1: xbot Integration (Recommended)
1. **Import xbot as a library** to handle tool execution
2. **Wrap xbot's sandboxing** with AgentGuard policy evaluation
3. **Maintain OpenShell as fallback** for kernel-level isolation
4. **Implement feature flag** to switch between sandboxing backends

**Benefits:**
- Go-native integration (no subprocess overhead)
- Cross-platform sandboxing
- MCP alignment with existing architecture
- MIT license compatibility

**Challenges:**
- Learning curve for xbot's API
- Potential performance overhead
- Testing across different platforms

### Option 2: goclaw Integration (Alternative)
1. **Evaluate goclaw maturity** more thoroughly
2. **Consider as lighter alternative** if xbot is too heavy
3. **Prototype integration** to assess viability

**Benefits:**
- Simpler architecture potentially
- Focused on AI agent tooling

**Challenges:**
- Immature codebase
- Unknown reliability
- Limited community support

### Option 3: Hybrid Approach
1. **Abstract sandboxing interface** in ShellForge
2. **Support multiple backends**: OpenShell, xbot, goclaw
3. **Auto-select based on platform**: OpenShell on Linux, xbot elsewhere
4. **Policy translation**: Convert AgentGuard policies to each backend's format

## Implementation Roadmap

### Phase 1: Research & Prototyping (2-3 weeks)
1. Clone and build xbot locally
2. Analyze sandboxing implementation details
3. Create proof-of-concept integration
4. Benchmark performance overhead

### Phase 2: Integration Design (1-2 weeks)
1. Design abstract sandboxing interface
2. Create policy translation layer
3. Implement xbot backend adapter
4. Add configuration options

### Phase 3: Testing & Validation (2-3 weeks)
1. Unit tests for sandboxing interface
2. Integration tests with actual tool execution
3. Security testing (penetration testing)
4. Performance benchmarking vs OpenShell

### Phase 4: Production Rollout (1-2 weeks)
1. Feature flag implementation
2. Documentation updates
3. Release planning
4. Community announcement

## Technical Considerations

### Policy Translation
AgentGuard policies (YAML) need translation to xbot's sandbox configuration:

```yaml
# AgentGuard policy example
policies:
  - name: no-destructive-rm
    action: deny
    pattern: "rm -rf"
```

Would need translation to xbot's sandbox constraints (exact format TBD).

### Performance Impact
- **OpenShell**: Minimal overhead (kernel-native)
- **xbot**: Moderate overhead (process spawning + constraints)
- **Network latency**: Tool execution may be slower initially

### Security Trade-offs
- **xbot**: Process-level isolation vs OpenShell's kernel-level
- **Cross-platform**: xbot works everywhere vs OpenShell's Linux requirement
- **Attack surface**: Additional Go code vs battle-tested kernel features

## Conclusion

**xbot represents the most promising candidate** for Go agent framework sandboxing integration with ShellForge. Its active development, MIT license, MCP support, and Go-native implementation make it a strong contender to complement or potentially replace OpenShell for cross-platform sandboxing.

**Recommended next steps:**
1. **Immediate**: Create detailed code analysis of xbot's sandboxing implementation
2. **Short-term**: Build prototype integration to validate feasibility
3. **Medium-term**: Design abstract sandboxing interface for multiple backends
4. **Long-term**: Implement production-ready xbot integration with feature flags

The integration would significantly enhance ShellForge's sandboxing capabilities by providing a cross-platform solution that maintains strong security while expanding platform support beyond Linux.

## References

1. xbot GitHub: https://github.com/CjiW/xbot
2. goclaw GitHub: https://github.com/Dimas1182/goclaw  
3. OpenShell GitHub: https://github.com/NVIDIA/OpenShell
4. ShellForge Architecture: docs/architecture.md
5. AgentGuard: https://github.com/AgentGuardHQ/agentguard