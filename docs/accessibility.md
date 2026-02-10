# Accessibility Guide

XelaNote implements comprehensive accessibility features to ensure the application is usable by everyone, including users who rely on assistive technologies like screen readers or keyboard-only navigation.

## Table of Contents

- [Overview](#overview)
- [Keyboard Navigation](#keyboard-navigation)
- [Screen Reader Support](#screen-reader-support)
- [Focus Management](#focus-management)
- [Accessible Dialogs](#accessible-dialogs)
- [Motion & Animation](#motion--animation)
- [ARIA Attributes](#aria-attributes)
- [Internationalization](#internationalization)
- [Testing Accessibility](#testing-accessibility)

---

## Overview

XelaNote follows the Web Content Accessibility Guidelines (WCAG 2.1) and implements the following key accessibility features:

- **Keyboard Navigation**: Full keyboard support with skip links
- **Focus Management**: Automatic focus trapping in dialogs and modals
- **Screen Reader Support**: Proper ARIA attributes and semantic HTML
- **Reduced Motion**: Respects `prefers-reduced-motion` system setting
- **Custom Accessible Dialogs**: Replaces native `confirm()` and `alert()` with fully accessible alternatives
- **Internationalized Labels**: All UI elements have translated accessible labels

---

## Keyboard Navigation

### Skip Link

XelaNote provides a skip link that becomes visible when focused with the Tab key. This allows keyboard users to skip directly to the main content without navigating through the entire sidebar and navigation menus.

**Implementation**: The skip link is visually hidden by default but appears when focused via keyboard navigation.

**Location**: `frontend/src/routes/+layout.svelte` (lines 248-254)

**CSS Classes**:
```css
.sr-only                    /* Visually hidden */
focus:not-sr-only           /* Visible on focus */
focus:absolute focus:top-2  /* Position when focused */
```

**Keyboard Shortcut**: Press `Tab` on page load to reveal the skip link, then press `Enter` to skip to main content.

---

## Screen Reader Support

### Toast Notifications

The Toast component uses proper ARIA roles and live regions to announce notifications to screen reader users:

```svelte
<div
  role="region"
  aria-label={$_('accessibility.notifications')}
  class="fixed bottom-4 right-4 z-50"
>
  <div
    role={type === 'error' ? 'alert' : 'status'}
    aria-live={type === 'error' ? 'assertive' : 'polite'}
    aria-atomic="true"
  >
    {message}
  </div>
</div>
```

**Features**:
- `role="alert"` for error messages (interrupts screen reader)
- `role="status"` for success/info messages (announced politely)
- `aria-live="assertive"` for errors, `aria-live="polite"` for other notifications
- `aria-atomic="true"` ensures entire message is read

**File**: `frontend/src/lib/components/Toast.svelte`

### Offline Banner

The offline banner uses an assertive live region to immediately notify users when they lose connectivity:

```svelte
<div
  role="alert"
  aria-live="assertive"
  class="fixed top-0 left-0 right-0 z-50 bg-amber-500"
>
  <WifiOff size={20} />
  <span>You're offline - Read-only mode</span>
</div>
```

**File**: `frontend/src/lib/components/OfflineBanner.svelte`

### Sidebar Navigation

All sidebar buttons include descriptive `aria-label` attributes for screen readers:

```svelte
<button aria-label={$_('page.sidebar.new_note')} title="New Note (Ctrl+N)">
  <FilePlus size={18} />
</button>

<button aria-label={$_('page.sidebar.search')} title="Search (Ctrl+K)">
  <Search size={18} />
</button>

<button aria-label={$_('page.sidebar.trash')} title="Trash">
  <Trash2 size={18} />
</button>
```

**File**: `frontend/src/lib/components/Sidebar.svelte`

**Features**:
- Every icon-only button has an `aria-label`
- All labels are internationalized using svelte-i18n
- Tooltips provide additional context with keyboard shortcuts

---

## Focus Management

### Focus Trap in Dialogs

XelaNote uses the `focus-trap` library to manage focus within dialogs and modals. This prevents keyboard users from accidentally tabbing out of a modal while it's open.

**Key Features**:
1. **Automatic Focus**: When a dialog opens, focus automatically moves to the first focusable element
2. **Tab Cycling**: Tab and Shift+Tab cycle through focusable elements within the dialog
3. **Focus Restoration**: When the dialog closes, focus returns to the element that opened it
4. **Keyboard Escape**: Pressing Escape closes the dialog (configurable)

**Implementation**: `frontend/src/lib/components/ui/BaseDialog.svelte`

```typescript
import { createFocusTrap, type FocusTrap } from 'focus-trap';

$effect(() => {
  if (open && dialogRef) {
    // Store previously focused element
    previousActiveElement = document.activeElement;

    // Create and activate focus trap
    focusTrap = createFocusTrap(dialogRef, {
      escapeDeactivates: false,  // Custom escape handling
      allowOutsideClick: true,
      fallbackFocus: dialogRef,
      returnFocusOnDeactivate: false  // Manual restoration
    });

    requestAnimationFrame(() => {
      focusTrap?.activate();
    });

    // Prevent body scroll
    document.body.style.overflow = 'hidden';

    return () => {
      // Cleanup on close
      focusTrap?.deactivate();
      document.body.style.overflow = '';

      // Restore focus
      if (previousActiveElement instanceof HTMLElement) {
        previousActiveElement.focus();
      }
    };
  }
});
```

---

## Accessible Dialogs

XelaNote replaces native `confirm()` and `alert()` dialogs with custom, fully accessible implementations.

### Why Replace Native Dialogs?

Native browser dialogs (`window.confirm()`, `window.alert()`) have several accessibility issues:
- Poor screen reader support
- Inconsistent keyboard navigation
- No customization of labels or styling
- Block the main thread (bad UX)

### Custom Dialog Components

#### BaseDialog

The foundation component for all dialogs, providing:
- Focus trap integration
- ARIA attributes (`role="dialog"`, `aria-modal="true"`, `aria-labelledby`)
- Keyboard support (Escape to close)
- Backdrop click handling
- Size variants (`sm`, `md`, `lg`, `xl`)
- Style variants (`default`, `danger`)

**File**: `frontend/src/lib/components/ui/BaseDialog.svelte`

#### ConfirmDialog

Replaces `window.confirm()` with an accessible confirmation dialog:

```typescript
import * as dialog from '$lib/stores/dialog.svelte';

// Old way (not accessible):
const result = window.confirm('Delete this note?');

// New way (accessible):
const result = await dialog.confirm({
  title: 'Confirm Action',
  message: 'Delete this note?',
  confirmText: 'Delete',
  cancelText: 'Cancel',
  variant: 'danger'  // Visual emphasis for destructive actions
});

if (result) {
  // User clicked "Delete"
} else {
  // User clicked "Cancel" or pressed Escape
}
```

**Features**:
- Returns a Promise (async/await support)
- Configurable button labels
- Danger variant for destructive actions
- Keyboard accessible (Tab, Enter, Escape)
- Focus trapped within dialog

**File**: `frontend/src/lib/components/ui/ConfirmDialog.svelte`

#### AlertDialog

Replaces `window.alert()` with an accessible alert dialog:

```typescript
import * as dialog from '$lib/stores/dialog.svelte';

// Old way (not accessible):
window.alert('Operation completed!');

// New way (accessible):
await dialog.alert({
  title: 'Success',
  message: 'Operation completed!',
  confirmText: 'OK',
  variant: 'default'  // or 'warning', 'danger'
});
```

**Features**:
- Returns a Promise that resolves when dismissed
- Three variants: `default` (info), `warning`, `danger`
- Contextual icons (Info, Warning, AlertCircle)
- Keyboard accessible
- Screen reader friendly

**File**: `frontend/src/lib/components/ui/AlertDialog.svelte`

### Dialog Store

The dialog system is managed by a centralized store that coordinates state and promises.

**File**: `frontend/src/lib/stores/dialog.svelte.ts`

**API**:
```typescript
// Show confirmation dialog
function confirm(options: ConfirmOptions): Promise<boolean>

// Show alert dialog
function alert(options: AlertOptions): Promise<void>

// Internal: resolve dialogs
function resolveConfirm(result: boolean): void
function resolveAlert(): void

// Get current state
function getConfirmState(): ConfirmOptions | null
function getAlertState(): AlertOptions | null
```

**Usage in Components**:

The ConfirmDialog and AlertDialog components are rendered globally in the root layout (`+layout.svelte`) and automatically appear when the dialog store state changes:

```svelte
<!-- frontend/src/routes/+layout.svelte -->
<ConfirmDialog />
<AlertDialog />
```

---

## Motion & Animation

XelaNote respects the user's motion preferences set at the operating system level via the `prefers-reduced-motion` media query.

### Implementation

When a user has enabled "Reduce Motion" in their OS accessibility settings:
- All animations are drastically reduced (0.01ms duration)
- Transitions are minimized
- Smooth scrolling is disabled
- Animation iterations are limited to 1

**File**: `frontend/src/app.css` (lines 1087-1097)

```css
@media (prefers-reduced-motion: reduce) {
  *,
  *::before,
  *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
    scroll-behavior: auto !important;
  }
}
```

### How to Test

**macOS**:
System Preferences → Accessibility → Display → Reduce motion

**Windows**:
Settings → Ease of Access → Display → Show animations in Windows

**Linux (GNOME)**:
```bash
gsettings set org.gnome.desktop.interface enable-animations false
```

---

## ARIA Attributes

XelaNote uses semantic HTML and ARIA attributes throughout the application:

### Dialog Components
- `role="dialog"` - Identifies the element as a dialog
- `aria-modal="true"` - Indicates the dialog is modal (focus trapped)
- `aria-labelledby="dialog-title"` - Links dialog to its title
- `aria-label="Close dialog"` - Describes close button action

### Notifications
- `role="region"` - Defines a landmark region
- `role="alert"` - For error messages (assertive)
- `role="status"` - For info/success messages (polite)
- `aria-live="assertive|polite"` - Controls announcement timing
- `aria-atomic="true"` - Ensures full message is announced
- `aria-label="Notifications"` - Labels notification container

### Navigation
- `role="presentation"` - For decorative elements
- `aria-label` - Descriptive labels for icon-only buttons
- `aria-hidden="true"` - Hides decorative/redundant content from screen readers

---

## Internationalization

All accessibility labels are fully internationalized and stored in locale files.

### Accessibility Keys

**File**: `frontend/src/lib/locales/en.json`

```json
{
  "accessibility": {
    "skip_to_main": "Skip to main content",
    "close_dialog": "Close dialog",
    "notifications": "Notifications",
    "close_notification": "Close notification"
  }
}
```

### Dialog Keys

```json
{
  "dialog": {
    "confirm_title": "Confirmation",
    "confirm": "Confirm",
    "cancel": "Cancel",
    "delete_note_confirm": "Really delete this note?",
    "create_missing_note": "Note does not exist. Create it?",
    "permanent_delete_confirm": "Permanently delete this note?",
    "empty_trash_confirm": "Empty trash? This cannot be undone.",
    "restore_note_confirm": "Restore this note?"
  }
}
```

### Sidebar Keys

All sidebar buttons have internationalized labels for both `aria-label` and tooltips:

```json
{
  "page": {
    "sidebar": {
      "new_note": "New note",
      "create_new_note": "Create new note",
      "new_folder": "New folder",
      "create_new_folder": "Create new folder",
      "search": "Search",
      "graph": "Graph",
      "trash": "Trash",
      "admin": "Admin",
      "settings": "Settings",
      "import_markdown": "Import Markdown",
      "export_markdown": "Export as Markdown",
      "collapse_sidebar": "Collapse sidebar",
      "expand_sidebar": "Expand sidebar",
      "close_drawer": "Close drawer"
    }
  }
}
```

### Usage in Components

```svelte
<script>
  import { _ } from 'svelte-i18n';
</script>

<button aria-label={$_('accessibility.close_dialog')}>
  <X size={18} />
</button>
```

---

## Testing Accessibility

### Keyboard Navigation Testing

1. **Tab Navigation**: Use Tab to move forward, Shift+Tab to move backward
2. **Skip Link**: Press Tab on page load to reveal skip link, Enter to activate
3. **Dialog Navigation**: Open a dialog, verify Tab cycles through focusable elements
4. **Escape Key**: Verify Escape closes dialogs and modals
5. **Enter Key**: Verify Enter activates buttons and confirms dialogs

### Screen Reader Testing

**Recommended Tools**:
- **NVDA** (Windows, free): https://www.nvaccess.org/
- **JAWS** (Windows, commercial): https://www.freedomscientific.com/products/software/jaws/
- **VoiceOver** (macOS/iOS, built-in): Cmd+F5 to activate
- **Orca** (Linux, free): Pre-installed on most distributions

**Test Checklist**:
- [ ] All buttons and links are announced with descriptive labels
- [ ] Notification messages are announced automatically
- [ ] Dialog titles and content are read when opened
- [ ] Form fields have associated labels
- [ ] Navigation landmarks are announced (sidebar, main content, etc.)

### Automated Testing Tools

**Browser Extensions**:
- **axe DevTools** (Chrome/Firefox): https://www.deque.com/axe/devtools/
- **WAVE** (Chrome/Firefox/Edge): https://wave.webaim.org/extension/
- **Lighthouse** (Chrome DevTools): Built-in, check Accessibility score

**Command Line**:
```bash
# Install axe-core CLI
npm install -g @axe-core/cli

# Run accessibility audit
axe http://localhost:5173 --tags wcag2a,wcag2aa
```

### Manual Testing Checklist

- [ ] Skip link appears on Tab and navigates to main content
- [ ] All interactive elements are keyboard accessible
- [ ] Focus is visible on all interactive elements
- [ ] Dialogs trap focus and restore focus on close
- [ ] Escape key closes dialogs
- [ ] ARIA labels are present and descriptive
- [ ] Color contrast meets WCAG AA standards (4.5:1 for text)
- [ ] Reduced motion preference is respected
- [ ] Screen reader announces notifications
- [ ] Error messages are clear and actionable

---

## Sidebar Redesign

The sidebar has been redesigned with accessibility in mind:

### New Layout

- **Top Section**: Notes tree with folder hierarchy
- **Bottom Section**: Compact icon bar with all action buttons

### Accessibility Features

1. **Icon Buttons**: All icon-only buttons have:
   - `aria-label` attribute for screen readers
   - Tooltip with keyboard shortcut on hover/focus
   - Sufficient touch target size (44x44px minimum on mobile)

2. **Keyboard Shortcuts**: Displayed in tooltips
   - New Note: `Ctrl+N` (Cmd+N on Mac)
   - Search: `Ctrl+K` (Cmd+K on Mac)
   - Settings: Visible in tooltip

3. **Responsive Icon Sizes**:
   - Mobile: 20px (larger touch targets)
   - Desktop: 18px (compact design)

4. **Visual Hierarchy**: Actions grouped logically:
   - Create (Note, Folder)
   - Navigate (Search, Graph, Trash)
   - System (Import, Export, Settings, Admin)
   - Account (Logout)

**File**: `frontend/src/lib/components/Sidebar.svelte`

---

## Component Reference

### New Components Added

| Component | File | Purpose |
|-----------|------|---------|
| **BaseDialog** | `frontend/src/lib/components/ui/BaseDialog.svelte` | Base dialog with focus-trap integration |
| **ConfirmDialog** | `frontend/src/lib/components/ui/ConfirmDialog.svelte` | Accessible confirmation dialog |
| **AlertDialog** | `frontend/src/lib/components/ui/AlertDialog.svelte` | Accessible alert dialog |
| **Dialog Store** | `frontend/src/lib/stores/dialog.svelte.ts` | State management for dialogs |

### Updated Components

| Component | File | Accessibility Improvements |
|-----------|------|----------------------------|
| **Toast** | `frontend/src/lib/components/Toast.svelte` | Added ARIA live regions and roles |
| **OfflineBanner** | `frontend/src/lib/components/OfflineBanner.svelte` | Added `role="alert"` and `aria-live="assertive"` |
| **Sidebar** | `frontend/src/lib/components/Sidebar.svelte` | Added aria-labels and tooltips to all buttons |
| **Layout** | `frontend/src/routes/+layout.svelte` | Added skip link, renders global dialogs |

---

## Future Improvements

Potential accessibility enhancements for future releases:

- [ ] High contrast theme for visually impaired users
- [ ] Configurable font size scaling
- [ ] Voice control support (Web Speech API)
- [ ] Landmark navigation shortcuts
- [ ] Focus indicators with increased visibility option
- [ ] Screen reader mode with optimized announcements
- [ ] Keyboard shortcut customization
- [ ] Dyslexia-friendly font option (OpenDyslexic)

---

## Resources

### WCAG Guidelines
- [WCAG 2.1 Overview](https://www.w3.org/WAI/WCAG21/quickref/)
- [ARIA Authoring Practices Guide](https://www.w3.org/WAI/ARIA/apg/)

### Testing Tools
- [axe DevTools](https://www.deque.com/axe/devtools/)
- [WAVE Browser Extension](https://wave.webaim.org/extension/)
- [Lighthouse Accessibility Audit](https://developers.google.com/web/tools/lighthouse)

### Screen Reader Resources
- [NVDA User Guide](https://www.nvaccess.org/files/nvda/documentation/userGuide.html)
- [VoiceOver User Guide](https://www.apple.com/voiceover/info/guide/)
- [JAWS Keyboard Shortcuts](https://www.freedomscientific.com/training/jaws/hotkeys/)

### Libraries Used
- [focus-trap](https://github.com/focus-trap/focus-trap) - Focus management for modals
- [svelte-i18n](https://github.com/kaisermann/svelte-i18n) - Internationalization

---

## Contributing

When contributing to XelaNote, please ensure all new features maintain accessibility standards:

1. **Add ARIA Labels**: All interactive elements must have descriptive labels
2. **Keyboard Support**: All functionality must be keyboard accessible
3. **Focus Management**: Maintain logical focus order and visibility
4. **Test with Screen Readers**: Verify announcements are clear and helpful
5. **Color Contrast**: Ensure text meets WCAG AA standards (4.5:1 ratio)
6. **Internationalize Labels**: Add all new labels to locale files
7. **Document Changes**: Update this guide when adding accessibility features

For questions or suggestions, please open an issue on GitHub.
