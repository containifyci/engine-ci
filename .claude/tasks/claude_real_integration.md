# Plan: Integrate Real Claude CLI with Mode Switching

## Current Analysis
- **Current State**: Using `scripts/claude-wrapper.sh` simulation script
- **Claude CLI Available**: `/Users/frank.ittermann@goflink.com/.nvm/versions/node/v24.4.0/bin/claude`
- **Key Challenge**: Claude CLI is interactive by default, needs adaptation for remote control
- **New Requirement**: Support switching between plan mode and auto-accept edits mode

## Implementation Strategy

### 1. Create Real Claude Wrapper Script
**Goal**: Replace simulation with actual Claude CLI integration
**Approach**: 
- Use `claude --print` for non-interactive responses
- Handle prompt/confirmation workflow through stdin/stdout pipes
- Maintain existing `PROMPT:` and `CONFIRM:` command protocol
- **NEW**: Support mode switching via command parameters

### 2. Implement Mode Switching System
**Plan Mode**:
- Use `claude --permission-mode plan` for planning-only responses
- Claude will create plans without executing any changes
- Remote interface shows plan details for user review
- User can approve/reject plans before execution

**Auto-Accept Mode**:
- Use `claude --permission-mode acceptEdits` for automatic execution
- Claude can make file changes, run commands, etc. automatically
- Higher efficiency but requires explicit user activation
- Clear safety warnings when enabling this mode

**Mode Detection & Control**:
- Add mode selection in web interface (radio buttons/toggle)
- Store current mode preference per session
- Clear visual indicators of current mode in mobile interface
- Mode switching persists across reconnections

### 3. Enhanced Web Interface for Mode Control
**Mode Selection UI**:
- Prominent mode toggle in mobile interface
- Visual indicators: 📋 Plan Mode vs ⚡ Auto-Execute Mode
- Confirmation dialog when switching to auto-execute
- Status display showing current mode at all times

**Safety Features**:
- Auto-execute mode requires explicit confirmation each session
- Clear warnings about auto-execute capabilities
- Option to review all planned changes before auto-execution
- Emergency "switch to plan mode" button always visible

### 4. Handle Claude CLI Interaction Patterns
**Print Mode Integration**:
- Use `claude --print --permission-mode [plan|acceptEdits]` based on current mode
- Parse Claude's response format and integrate with log streaming
- Handle timeout scenarios (Claude can sometimes take long to respond)

**Interactive Session Management**:
- Explore using `claude --continue` for conversation continuity
- Implement session management to maintain context across prompts
- Handle Claude's permission dialogs and user confirmations properly
- **NEW**: Pass mode parameters to maintain consistency

### 5. Security & Permission Configuration
**Safe Defaults**:
- **Default to plan mode** for maximum safety
- Configure appropriate `--allowedTools` for safe remote operation
- Use `--add-dir` to limit Claude's file access scope
- Implement timeout handling for Claude operations
- **Never use `--dangerously-skip-permissions`** (as requested)

**Mode-Specific Security**:
- Plan mode: Read-only operations, no file modifications
- Auto-execute mode: Full Claude capabilities with explicit user consent
- Clear audit trails of mode switches and operations
- Session timeout with automatic revert to plan mode

### 6. Enhanced Remote Control Features
**Bidirectional Communication**:
- Map remote "approve/reject" to Claude's internal confirmation system
- Implement proper Claude session initialization and cleanup
- Handle Claude's tool permission requests through remote interface
- **NEW**: Mode-aware command processing

**Mode-Aware Workflows**:
- Plan mode: Show plans, require explicit execution approval
- Auto-execute mode: Stream real-time execution progress
- Clear differentiation in logs between planning and execution
- Ability to interrupt auto-execution and switch to plan mode

### 7. API Enhancements
**New Endpoints**:
- `POST /mode` - Switch between plan/auto-execute modes
- `GET /mode` - Get current mode status
- `POST /execute-plan` - Execute a previously generated plan (plan mode only)

**Enhanced Existing Endpoints**:
- `/prompt` - Include current mode in processing
- `/status` - Show current Claude mode and capabilities
- `/logs` - Differentiate between plan logs and execution logs

### 8. Testing & Validation
**Test Scenarios**:
- Mode switching functionality
- Plan generation and execution workflows
- Auto-execute mode safety and controls
- Error conditions and recovery in both modes
- Session persistence across mode changes
- Performance with real Claude vs simulation

## Implementation Steps

1. **Create `scripts/claude-real.sh`** - Real Claude integration wrapper with mode support
2. **Update Claude Manager** - Add mode detection, switching, and Claude CLI integration
3. **Enhance Web Interface** - Add mode controls and visual indicators
4. **Add Mode API Endpoints** - Backend support for mode switching
5. **Implement Safety Controls** - Warnings, confirmations, and audit trails
6. **Update Error Handling** - Handle Claude-specific errors and mode-related issues
7. **Comprehensive Testing** - Test both modes with real Claude sessions
8. **Update Documentation** - Usage instructions for both modes and safety guidelines

## Benefits
- **Real Claude Power**: Access to actual Claude capabilities vs simulation
- **Flexible Control**: Choose between safe planning and efficient auto-execution
- **Tool Integration**: Claude's file editing, web search, and analysis tools
- **Safety First**: Plan mode prevents unwanted changes, auto-execute for trusted workflows
- **Production Ready**: True remote Claude control for real development work

## Safety Considerations
- **Default to plan mode** for maximum safety
- Clear visual indicators and warnings for mode switching
- Explicit confirmation required for auto-execute mode
- Timeout handling to prevent runaway Claude operations
- Clear audit trails of all Claude operations and mode changes
- Emergency controls to immediately switch to plan mode
- Session-based mode persistence (doesn't persist across app restarts)
- Option to revert to simulation mode for testing/demo purposes

## Mobile Interface Enhancements
- **Mode Toggle**: Prominent toggle switch with clear labels
- **Visual Indicators**: Color-coded mode indicators (blue for plan, orange for auto-execute)
- **Safety Warnings**: Modal dialogs when enabling auto-execute mode
- **Status Cards**: Current mode displayed in status section
- **Quick Actions**: One-tap mode switching with confirmation

## Phase 1: Foundation Setup
### Tasks
1. Create `scripts/claude-real.sh` wrapper script
2. Add mode detection to Claude Manager
3. Test basic Claude CLI integration
4. Implement safety defaults and error handling

## Phase 2: Mode Switching Implementation
### Tasks
1. Add mode switching logic to backend
2. Create mode API endpoints
3. Implement session-based mode persistence
4. Add comprehensive error handling for both modes

## Phase 3: Enhanced Web Interface
### Tasks
1. Add mode selection UI components
2. Implement visual indicators and status displays
3. Add safety warnings and confirmation dialogs
4. Update real-time log streaming for mode-aware display

## Phase 4: Testing & Documentation
### Tasks
1. Comprehensive testing with real Claude sessions
2. Validate safety controls and mode switching
3. Update documentation with new features
4. Performance testing and optimization