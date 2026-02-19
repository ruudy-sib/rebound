# Rebound Retry Skill

A Claude Code skill for implementing intelligent retry orchestration using Rebound.

## What is This?

This skill helps developers integrate Rebound into their Go applications for resilient failure handling with:
- Exponential backoff retry logic
- Dead letter queue support
- Kafka and HTTP destination support
- Production-ready patterns

## When to Use This Skill

Invoke this skill when you need to:
- Add retry logic to your application
- Handle transient failures gracefully
- Implement exponential backoff
- Set up dead letter queues
- Build resilient microservices
- Integrate with Kafka or HTTP webhooks

## Activation Keywords

The skill activates on mentions of:
- `retry`, `rebound`, `exponential backoff`
- `dead letter queue`, `dlq`
- `failure handling`, `resilience`
- `kafka retry`, `http retry`

## Usage

### Automatic Activation

Simply mention retry-related terms in your request:

```
"Add retry logic to the order service using Rebound"
"Implement exponential backoff for failed webhook deliveries"
"Set up a dead letter queue for failed Kafka messages"
```

### Manual Invocation

```bash
/implement-rebound-retry
```

## What the Skill Provides

1. **Quick Start Code** - Basic Rebound integration
2. **Integration Patterns** - HTTP handler, Kafka consumer, DI patterns
3. **Retry Strategies** - Configuration for different use cases
4. **Best Practices** - Production-ready recommendations
5. **Performance Guidance** - Based on real benchmarks
6. **Troubleshooting** - Common issues and solutions

## File Structure

```
implement-rebound-retry/
├── skill.md         # Main skill documentation
├── REFERENCE.md     # Advanced patterns and examples
└── README.md        # This file
```

## Examples Included

- Basic integration
- HTTP handler integration
- Kafka consumer integration
- Dependency injection pattern
- Adaptive retry strategies
- Multi-tenant configuration
- Batch operations
- Monitoring and observability

## Performance Reference

Based on benchmarks (Apple M4 Pro):
- Sequential: 4,700 tasks/sec
- Parallel: 27,600 tasks/sec
- Latency: 214 μs per task
- Memory: 1.4 KB per task

## Related Skills

- `implement-go-kafka-consumer` - Kafka consumer patterns
- `implement-go-redis-client` - Redis integration
- `implement-go-dependency-injection` - DI patterns

## Repository

The Rebound package is in: `rebound/pkg/rebound`

See full documentation in: `pkg/rebound/README.md`

## Adding to Claude Code

To make this skill available globally:

1. Copy to Claude plugins directory:
   ```bash
   cp -r .claude/skills/implement-rebound-retry \
     ~/.claude/plugins/{your-plugin}/skills/
   ```

2. Or create as a standalone plugin:
   ```bash
   # Use create-claude-code-plugin skill
   /create-claude-code-plugin
   ```

## Contributing

To improve this skill:
1. Update `skill.md` with new patterns
2. Add examples to `REFERENCE.md`
3. Test with real-world scenarios
4. Submit feedback

## Support

For questions or issues:
- Check `pkg/rebound/README.md`
- Review examples in `examples/`
- Open an issue on GitHub
