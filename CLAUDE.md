# Go OPRF Project - Planning Methodology

## Project Overview
Porting liboprf C library to pure Go with byte-compatible output. Source: ~/projects/git/liboprf

## Planning Structure

### Hierarchical Plan Documents

All plans live in `plan/` directory.

#### plan/PLAN-1.md (Top Level)
Contains high-level steps for the entire project. Each step is a major phase.

**Structure:**
- Step 1, Step 2, Step 3, etc.
- Each step has deliverables
- Minimal "Notes for Later Steps" section (only critical cross-step information)

#### plan/PLAN-1-N.md (Detailed Plans)
Each major step from PLAN-1 gets its own detailed plan document.

**Naming:** plan/PLAN-1-1.md, plan/PLAN-1-2.md, plan/PLAN-1-3.md, etc.

**Structure:**
- Goal and prerequisites
- Detailed tasks (Task N.1, N.2, N.3, etc.)
- Each task has:
  - Action items
  - Success criteria
  - Time estimate
- Total time estimate
- Completion criteria checklist
- Dependencies for next step

#### plan/PLAN-1-N-M.md (Sub-Plans, if needed)
If a detailed plan is still too large (>1 day), break it down further.

**Example:** plan/PLAN-1-2-1.md for detailed breakdown of Step 2, Task 1

**Rule:** Continue breaking down until a plan represents ~1 day of work (4-8 hours)

### Supporting Documents

#### C-API-ANALYSIS.md
Documents understanding of C implementation:
- Function signatures and purposes
- Data structures
- Constants
- Libsodium → Go mappings
- Implementation notes

**Cleanup:** Once implementation is complete and tests pass, archive or delete this file.

#### TEST-VECTORS.md
Extracted test vectors from C implementation:
- Test name
- Input values (hex)
- Keys (hex)
- Expected outputs (hex)
- Test description

**Cleanup:** Once all test vectors are incorporated into test files, delete this file.

#### plan/NOTES.md (Temporary)
Running notes during current session:
- Decisions made
- Issues encountered
- Solutions applied

**Cleanup:** At end of each session:
- Delete resolved notes
- Move important decisions into code comments or relevant PLAN file
- Keep this file minimal - ideally empty at session end

## Workflow for Each Session

### Starting a Session
1. Read `plan/PLAN-1.md` to understand overall status
2. Identify current step number
3. Read corresponding `plan/PLAN-1-N.md` for detailed tasks
4. Review `plan/NOTES.md` for any critical previous session notes (should be minimal)
5. Create TodoWrite list for current tasks

### During Work
1. Follow tasks in current `plan/PLAN-1-N.md` sequentially
2. Check off completion criteria as tasks finish
3. Add brief notes to `plan/NOTES.md` only if critical for current session
4. Update TodoWrite list as tasks complete
5. Put important decisions in code comments, not separate notes

### Completing a Step
1. Verify all completion criteria checked
2. Update `plan/PLAN-1.md` status
3. Clean up `plan/NOTES.md`:
   - Delete resolved items
   - Move critical info to appropriate location (code comments, next PLAN file)
4. Create `plan/PLAN-1-(N+1).md` for next step (if not already exists)
5. Tidy everything up:
   - Clear TodoWrite list or mark all as completed
   - Ensure all changes are committed
   - Verify tests pass
   - Check for any temporary files or uncommitted work
6. **Inform user**: Explicitly let the user know tidying is complete and we're ready for the next step

### When a Plan is Too Large
If tasks in `plan/PLAN-1-N.md` exceed 8 hours:
1. Create `plan/PLAN-1-N-1.md`, `plan/PLAN-1-N-2.md`, etc.
2. Break tasks into smaller chunks
3. Ensure each sub-plan is ≤1 day of work

## Key Principles

### Keep It Minimal
- Delete notes as work completes
- Move completed plans to plan/archive/ if needed for reference
- Keep only current work visible
- Put decisions in code comments or documentation, not separate files

### Documentation is Present/Future Tense
- Documents describe current state (present tense)
- Or target state (future tense)
- Never conversational updates about what just happened

### Incremental Progress
- Complete one task fully before moving to next
- Run tests after each modification
- Commit working code frequently

### Cross-Verification
- Every ported function must have test with C test vector
- Byte-for-byte output compatibility required
- Test on multiple platforms

### Notes for Future Steps
When working on early steps, document information needed by later steps:
- Add to "Notes for Later Steps" section in `plan/PLAN-1.md` (keep minimal)
- Or add to specific `plan/PLAN-1-N.md` if known which step needs it
- Delete notes once they're acted upon

## Success Criteria for Entire Project
From INSTRUCTIONS.md:
- [ ] All core OPRF functions ported from oprf.c
- [ ] All threshold OPRF functions ported from toprf.c
- [ ] All test vectors from C implementation pass in Go
- [ ] Cross-platform tests pass (Linux, Mac, Windows minimum)
- [ ] Code review by cryptography-aware developer
- [ ] No CGo dependencies
- [ ] Documentation complete

## Current Status
**Current Step:** Step 1 - Project Setup and C Code Analysis
**Current Plan:** plan/PLAN-1-1.md
**Next Plan:** plan/PLAN-1-2.md (to be created after Step 1 completes)

## File Organization
```
go-oprf/
├── CLAUDE.md                  # This file - planning methodology
├── INSTRUCTIONS.md            # Original porting instructions (reference only)
├── plan/                      # All planning documents
│   ├── PLAN-1.md             # High-level plan
│   ├── PLAN-1-1.md           # Step 1 detailed plan
│   ├── PLAN-1-2.md           # Step 2 detailed plan (to be created)
│   ├── NOTES.md              # Temporary session notes (keep minimal)
│   └── archive/              # Completed plans (create when needed)
├── C-API-ANALYSIS.md         # C code analysis (temporary, delete when done)
├── TEST-VECTORS.md           # Test vectors (temporary, delete when in tests)
├── oprf/                     # Basic OPRF package
├── toprf/                    # Threshold OPRF package
└── internal/utils/           # Shared utilities
```

## Tips for Long-Term Success
1. **Read before coding**: Understand C code thoroughly first
2. **Test immediately**: Write test for each function before moving on
3. **Document decisions**: Future you needs to know why you did things
4. **Stay aligned**: Keep structure close to C implementation
5. **Verify constantly**: Run all tests after each change
6. **One day at a time**: Break work into manageable chunks
