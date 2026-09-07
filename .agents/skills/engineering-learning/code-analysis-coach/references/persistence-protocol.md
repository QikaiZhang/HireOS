# Persistence Protocol

## Eligibility And Paths

Persist only when the user explicitly asks for notes, a learning record, or interview material. Create no document for L1 by default. Agree on the artifact before writing it. Use these paths exactly:

- `learning-docs/code-analysis-coach/analysis/01_模块鸟瞰_{模块}.md`: A useful module overview.
- `learning-docs/code-analysis-coach/analysis/02_业务流程_{模块}.md`: A source-confirmed flow worth recovering later.
- `learning-docs/code-analysis-coach/analysis/03_关键设计_{模块}.md`: An L2/L3 design with meaningful reasoning.
- `learning-docs/code-analysis-coach/analysis/04_面试素材_{模块}.md`: An L3/L4 interview artifact.
- `learning-docs/code-analysis-coach/self_improvement/todo_{YYYYMMDD}_{模块}.md`: A concrete gap or follow-up worth revisiting.

Use a stable, filesystem-safe module identifier. Do not create empty folders or placeholder documents.

## Writing Protocol

State the proposed filename and what it will preserve in one sentence. After the user agrees, write the complete standalone Markdown document to that exact path and link it in the response. Do not emit `FILE_PATH`/`FILE_CONTENT` envelopes in chat.

## Document Quality

Start from the closest template and delete unsupported sections. Include, where evidence exists: background, current implementation, confirmed flow, key design and rationale, alternatives, risks, and interview focus. Mark gaps as "待确认" and identify the missing artifact. Keep raw code excerpts short and only when they preserve an important contract or invariant.

Use `module-analysis.md` for overview or flow, `engineering-design.md` for L2/L3 reasoning, `interview-material.md` for L3/L4 narratives, and `self-review.md` for a specific learning gap. A document must remain useful without chat history.
