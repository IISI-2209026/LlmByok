## ADDED Requirements

### Requirement: Tech Dark Theme and Glassmorphism
The system SHALL render the official website using a Tech Dark theme and glassmorphism styling without using frontend frameworks.

#### Scenario: Visual presentation
- **WHEN** user loads the page
- **THEN** the system displays a dark background with fluorescent accents and semi-transparent cards with a blurred backdrop.

### Requirement: Terminal Typing Animation
The system SHALL display a terminal typing animation in the Hero section.

#### Scenario: Hero section animation
- **WHEN** user views the Hero section
- **THEN** the system simulates a terminal typing the command "byok launch copilot".

### Requirement: Navigation Bar
The system SHALL display a sticky navigation bar at the top of the page with quick-jump links to each section.

#### Scenario: Navigation links
- **WHEN** user clicks a navigation link (e.g., "特色", "如何運作", "核心功能", "上手", "安裝")
- **THEN** the page smoothly scrolls to the corresponding section.

### Requirement: Before and After Comparison
The system SHALL display a Before and After comparison in the Problem section.

#### Scenario: Problem demonstration
- **WHEN** user views the Problem section
- **THEN** the system shows the contrast between manual environment variable exports and the single "byok" command.

### Requirement: How It Works Flow Diagram
The system SHALL display a three-step flow diagram in the How It Works section with equally-sized step boxes.

#### Scenario: Flow diagram display
- **WHEN** user views the How It Works section
- **THEN** the system shows three equally-wide step cards connected by arrows, representing Profile management, byok launch, and tool startup.

### Requirement: Quick Start Section
The system SHALL display a "三分鐘上手" (Quick Start) section between Features and Install with a two-step tutorial.

#### Scenario: Step 1 — Configure Profile
- **WHEN** user views the Quick Start Step 1
- **THEN** the system shows the command `byok config add my-profile` with a copy button.

#### Scenario: Step 2 — Launch Tools
- **WHEN** user views the Quick Start Step 2
- **THEN** the system shows five cards (Copilot CLI, Codex CLI, Codex App, Claude, Pi), each with an official SVG icon and a launch command with a copy button.

### Requirement: Tool Icons
The system SHALL display official SVG icons for each supported tool instead of emoji.

#### Scenario: Icon display
- **WHEN** user views the Quick Start tool cards
- **THEN** each card displays the tool's official SVG icon (GitHub Copilot, OpenAI, Anthropic, Pi).

### Requirement: Install Section with OS Tabs
The system SHALL display the Install section at the bottom of the page with a "no dependencies" badge and OS-specific tab switching.

#### Scenario: No dependencies badge
- **WHEN** user views the Install section
- **THEN** the system displays a badge stating no runtime dependencies are required.

#### Scenario: OS tab switching
- **WHEN** user clicks an OS tab (Linux, macOS, or Windows)
- **THEN** the system shows the installation command for the selected platform and hides the others.

### Requirement: Copy to Clipboard Functionality
The system SHALL provide copy buttons with SVG icons (not text) for all code examples and installation commands.

#### Scenario: Successful copy
- **WHEN** user clicks the copy button next to a command
- **THEN** the system copies the command text to the clipboard and switches the button icon to a checkmark for 2 seconds.

#### Scenario: Copy unsupported fallback
- **WHEN** user clicks the copy button and the browser does not support the clipboard API
- **THEN** the system SHALL NOT throw a JavaScript exception.

### Requirement: Single-Line Command Display
The system SHALL display all command examples on a single line without horizontal scrollbars.

#### Scenario: Long command display
- **WHEN** a command example is longer than the container width
- **THEN** the system truncates the visible text with ellipsis and does not show a horizontal scrollbar.

