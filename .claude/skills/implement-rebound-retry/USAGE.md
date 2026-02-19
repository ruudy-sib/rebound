# How to Use the Rebound Retry Skill

## Quick Start

### Option 1: Manual Invocation (If Available Globally)

```bash
# Invoke the skill directly
/implement-rebound-retry
```

### Option 2: Automatic Activation

Simply mention retry-related keywords in your request, and Claude will automatically use this skill:

**Example requests:**
- "Add Rebound retry logic to my order service"
- "Implement exponential backoff for failed API calls"
- "Set up a dead letter queue for Kafka messages"
- "How do I integrate Rebound into my HTTP handler?"

## Testing the Skill Locally

Since the skill is in your project directory, test it by:

1. **Ask Claude about retry implementation:**
   ```
   "How do I implement retry logic using Rebound for my payment service?"
   ```

2. **Request specific patterns:**
   ```
   "Show me how to integrate Rebound with a Kafka consumer"
   ```

3. **Ask for configuration help:**
   ```
   "What retry strategy should I use for critical operations?"
   ```

## Making the Skill Globally Available

### Method 1: Add to Your User Skills Directory

```bash
# Create user skills directory if it doesn't exist
mkdir -p ~/.claude/skills

# Copy the skill
cp -r .claude/skills/implement-rebound-retry ~/.claude/skills/

# Restart Claude Code or reload
```

### Method 2: Create as a Plugin

Use the `create-claude-code-plugin` skill to package this as a plugin:

```bash
# This will guide you through plugin creation
/create-claude-code-plugin
```

Include the skill in your plugin structure:
```
my-rebound-plugin/
├── plugin.json
└── skills/
    └── implement-rebound-retry/
        ├── skill.md
        ├── REFERENCE.md
        └── README.md
```

## What You Get

When Claude uses this skill, you'll receive:

✅ **Quick Start Code** - Ready-to-use Rebound integration  
✅ **Integration Patterns** - HTTP, Kafka, DI examples  
✅ **Retry Strategies** - Configured for your use case  
✅ **Best Practices** - Production recommendations  
✅ **Performance Guidance** - Based on real benchmarks  
✅ **Troubleshooting Help** - Common issues and fixes  

## Example Conversations

### Example 1: Basic Integration

**You:** "I need to add retry logic to my email service. It sometimes fails when the SMTP server is unavailable."

**Claude (using this skill):** Will provide:
- Basic Rebound setup code
- Retry strategy for transient network errors
- Configuration for email use case
- Dead letter queue setup

### Example 2: Kafka Consumer

**You:** "How do I integrate Rebound with my Kafka consumer?"

**Claude (using this skill):** Will show:
- Kafka consumer with Rebound integration
- Message retry scheduling
- DLQ routing
- Error handling patterns

### Example 3: Performance Optimization

**You:** "I need to create 10,000 retry tasks per second. Will Rebound handle that?"

**Claude (using this skill):** Will explain:
- Parallel task creation (27K+ tasks/sec capable)
- Performance characteristics
- Optimization techniques
- Batch operation patterns

## Skill Capabilities

The skill can help with:

### ✅ Implementation
- Basic Rebound integration
- HTTP handler patterns
- Kafka consumer patterns
- Dependency injection setup

### ✅ Configuration
- Retry strategies by use case
- Exponential backoff tuning
- Dead letter queue setup
- Multi-environment config

### ✅ Advanced Patterns
- Adaptive retry strategies
- Priority-based retry
- Multi-tenant configuration
- Batch operations

### ✅ Operations
- Monitoring setup
- Performance tuning
- Troubleshooting
- Testing patterns

## Customizing the Skill

You can customize the skill for your needs:

1. **Edit skill.md** - Add your specific patterns
2. **Update REFERENCE.md** - Add advanced examples
3. **Modify activation keywords** - In the frontmatter description

Example customization:
```yaml
---
name: implement-rebound-retry
description: "... Activates on: retry, rebound, YOUR_CUSTOM_KEYWORDS"
---
```

## Skill vs Subagent

### Use a Skill when:
- Task is focused and deterministic
- You want quick code generation
- Pattern-based responses work well
- Examples and templates are sufficient

### Use a Subagent when:
- Need conversational interaction
- Complex decision-making required
- Multiple rounds of refinement needed
- Custom analysis per project

**For Rebound, a skill is recommended** because integration patterns are well-defined and example-driven.

## Creating a Subagent Alternative

If you prefer a conversational subagent instead:

```bash
# Use the create subagent skill
/create-claude-code-subagent
```

Configure it to:
- Name: `rebound-integration-expert`
- Role: Help with Rebound retry orchestration
- Tools: Read, Edit, Write, Bash, Grep, Glob
- Activation: "rebound, retry logic, exponential backoff"

## Testing

Test the skill by asking questions like:

```
1. "Show me a basic Rebound setup"
2. "How do I configure retry strategies?"
3. "What's the performance of Rebound?"
4. "Help me integrate Rebound with my Kafka consumer"
5. "I need to implement a dead letter queue"
```

## Troubleshooting

**Skill not activating?**
- Check activation keywords in description
- Try manual invocation: `/implement-rebound-retry`
- Verify file location: `.claude/skills/implement-rebound-retry/`

**Need more examples?**
- Check `REFERENCE.md` for advanced patterns
- See project examples in `examples/`
- Review benchmarks in `examples/07-consumer-benchmark/`

## Next Steps

1. ✅ Skill is created in `.claude/skills/implement-rebound-retry/`
2. Test it by asking retry-related questions
3. Customize for your needs
4. Share with your team (via plugin)
5. Contribute improvements back

## Support

- **Skill Issues**: Open an issue in your project
- **Rebound Issues**: Check `pkg/rebound/README.md`
- **Claude Code Issues**: See `/help`
