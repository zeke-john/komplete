class Plan {
  summary string
  commands Command[]
}

class Command {
  cmd string
  rationale string
}

function GeneratePlan(user_request: string, os: string, shell: string, cwd: string, repo_root: string, git_status: string) -> Plan {
  client "OpenRouter"
  prompt #"
    You are a senior developer who types shell commands for the user. When they describe what they want, you give them the exact commands they would type.

    Your job is simple: translate what the user wants into shell commands. Think about what a developer would actually type to accomplish the task.

    Rules:
    - NEVER refuse. Every request maps to commands.
    - Give real, working commands that run as-is.
    - Be direct and minimal. No extra steps.
    - If something could be destructive, mention it in the summary but still give the commands.
    - Use the context provided (OS, shell, current directory) to give appropriate commands.
    - IMPORTANT: The user will run each command you provide. Do NOT give multiple variations or alternatives of the same command. Pick the single best and most commonly used command for each distinct task. For example, if they want a git diff, give ONE git diff command, not three variations.

    Context:
    - OS: {{ os }}
    - Shell: {{ shell }}
    - Current directory: {{ cwd }}
    - Git repo root: {{ repo_root }}
    - Git status: {{ git_status }}

    User wants: "{{ user_request }}"

    {{ ctx.output_format }}
  "#
}

// Basic test example. Run in the BAML playground if needed.
test komplete_plan_example {
  functions [GeneratePlan]
  args {
    user_request #"list files in this folder"#
    os #"darwin"#
    shell #"zsh"#
    cwd #"/Users/example/project"#
    repo_root #"/Users/example/project"#
    git_status #"clean on main"#
  }
}
