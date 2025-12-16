# Catalyst UI Kit Port Plan

## Executive Summary

This document outlines the comprehensive plan for porting the Catalyst UI Kit (React + Headless UI + Tailwind CSS) to our Go templates + htmx + Alpine.js architecture. The goal is to create a themeable, production-ready component library that can be rapidly restyled for future projects.

**Key decisions:**
- CSS custom properties for all theming (colors, spacing, shadows, radii)
- Alpine.js for client-side interactivity (dropdowns, dialogs, tabs)
- htmx for server-driven interactions (form submissions, live search)
- Go template partials for component encapsulation
- Dark mode support via class strategy with CSS variables

---

## 1. Component Library Architecture

### 1.1 Directory Structure

```
web/
  templates/
    components/                    # Component library
      _variables.css               # CSS custom properties (imported in input.css)

      # Primitive components (no dependencies)
      primitives/
        button.html
        link.html
        badge.html
        avatar.html
        divider.html
        text.html
        heading.html
        code.html

      # Form components
      forms/
        input.html
        textarea.html
        select.html
        checkbox.html
        radio.html
        switch.html
        fieldset.html              # Field, FieldGroup, Label, Description, ErrorMessage
        input-group.html           # Input with leading/trailing icons

      # Interactive components (require Alpine.js)
      interactive/
        dropdown.html
        dialog.html
        alert.html                 # Modal alert/confirm dialog
        listbox.html
        combobox.html

      # Data display components
      data/
        table.html
        description-list.html
        pagination.html

      # Navigation components
      navigation/
        navbar.html
        sidebar.html
        sidebar-layout.html
        stacked-layout.html

    layouts/
      app.html                     # Updated to use new component system
      auth.html
      public.html

    pages/
      ...

  static/
    css/
      input.css                    # Main CSS entry point
      themes/
        default.css                # Default Lukaut theme (forest/gold)
        neutral.css                # Example alternative theme
    js/
      alpine-components.js         # Reusable Alpine.js component definitions
```

### 1.2 Component Organization Pattern

Each component template follows this pattern:

```html
{{/*
  Component: Button

  Variants: solid (default), outline, plain
  Colors: primary, secondary, danger, neutral (for solid variant only)
  Sizes: sm, md (default), lg

  Usage:
    {{template "button" dict "text" "Save" "variant" "solid" "color" "primary"}}
    {{template "button" dict "text" "Cancel" "variant" "outline" "href" "/back"}}
    {{template "button" dict "text" "Delete" "variant" "solid" "color" "danger" "disabled" true}}
*/}}

{{define "button"}}
{{- $variant := or .variant "solid" -}}
{{- $color := or .color "primary" -}}
{{- $size := or .size "md" -}}
{{- $disabled := or .disabled false -}}
{{- $type := or .type "button" -}}

{{if .href}}
<a href="{{.href}}"
   class="btn btn-{{$variant}} {{if eq $variant "solid"}}btn-{{$color}}{{end}} btn-{{$size}} {{.class}}"
   {{if $disabled}}aria-disabled="true" tabindex="-1"{{end}}>
  {{if .icon}}{{template "icon" dict "name" .icon "class" "btn-icon"}}{{end}}
  {{.text}}
</a>
{{else}}
<button type="{{$type}}"
        class="btn btn-{{$variant}} {{if eq $variant "solid"}}btn-{{$color}}{{end}} btn-{{$size}} {{.class}}"
        {{if $disabled}}disabled{{end}}
        {{if .hxPost}}hx-post="{{.hxPost}}"{{end}}
        {{if .hxTarget}}hx-target="{{.hxTarget}}"{{end}}
        {{if .hxSwap}}hx-swap="{{.hxSwap}}"{{end}}>
  {{if .icon}}{{template "icon" dict "name" .icon "class" "btn-icon"}}{{end}}
  {{.text}}
</button>
{{end}}
{{end}}
```

### 1.3 Naming Conventions

| Category | Convention | Examples |
|----------|------------|----------|
| Template definitions | kebab-case | `button`, `input-group`, `dropdown-menu` |
| CSS classes (components) | BEM-like | `.btn`, `.btn-solid`, `.btn-primary`, `.btn--loading` |
| CSS classes (utilities) | Tailwind standard | `mt-4`, `text-lg`, `flex` |
| CSS variables | kebab-case with prefix | `--color-primary`, `--spacing-4`, `--radius-lg` |
| Alpine data | camelCase | `x-data="{ isOpen: false, selectedItem: null }"` |
| Go template params | camelCase | `.variant`, `.isDisabled`, `.hxPost` |

---

## 2. CSS Variables Theming System

### 2.1 Core Design Tokens

Create `/workspaces/lukaut/web/static/css/themes/tokens.css`:

```css
:root {
  /* ========================================
     COLOR SYSTEM
     ======================================== */

  /* Semantic colors - these are what components reference */
  --color-primary: var(--color-forest-800);
  --color-primary-hover: var(--color-forest-700);
  --color-primary-active: var(--color-forest-900);
  --color-primary-subtle: var(--color-forest-50);

  --color-secondary: var(--color-clay-600);
  --color-secondary-hover: var(--color-clay-500);
  --color-secondary-subtle: var(--color-clay-50);

  --color-accent: var(--color-gold-300);
  --color-accent-hover: var(--color-gold-400);
  --color-accent-text: var(--color-forest-800);

  --color-danger: var(--color-red-600);
  --color-danger-hover: var(--color-red-500);
  --color-danger-subtle: var(--color-red-50);

  --color-success: var(--color-green-600);
  --color-success-subtle: var(--color-green-50);

  --color-warning: var(--color-amber-500);
  --color-warning-subtle: var(--color-amber-50);

  --color-info: var(--color-blue-600);
  --color-info-subtle: var(--color-blue-50);

  /* Surface colors */
  --color-background: var(--color-cream);
  --color-surface: white;
  --color-surface-raised: white;
  --color-surface-overlay: rgba(255, 255, 255, 0.75);

  /* Text colors */
  --color-text-primary: var(--color-zinc-950);
  --color-text-secondary: var(--color-zinc-500);
  --color-text-tertiary: var(--color-zinc-400);
  --color-text-inverse: white;
  --color-text-on-primary: white;
  --color-text-on-accent: var(--color-forest-800);

  /* Border colors */
  --color-border: var(--color-zinc-950-10);
  --color-border-hover: var(--color-zinc-950-20);
  --color-border-focus: var(--color-primary);
  --color-border-error: var(--color-danger);

  /* ========================================
     PALETTE - Raw color values
     These match Tailwind's zinc scale + brand colors
     ======================================== */

  --color-zinc-50: #fafafa;
  --color-zinc-100: #f4f4f5;
  --color-zinc-200: #e4e4e7;
  --color-zinc-300: #d4d4d8;
  --color-zinc-400: #a1a1aa;
  --color-zinc-500: #71717a;
  --color-zinc-600: #52525b;
  --color-zinc-700: #3f3f46;
  --color-zinc-800: #27272a;
  --color-zinc-900: #18181b;
  --color-zinc-950: #09090b;

  /* Zinc with opacity */
  --color-zinc-950-5: rgba(9, 9, 11, 0.05);
  --color-zinc-950-10: rgba(9, 9, 11, 0.1);
  --color-zinc-950-15: rgba(9, 9, 11, 0.15);
  --color-zinc-950-20: rgba(9, 9, 11, 0.2);

  /* Brand: Forest (Primary) */
  --color-forest-50: #E8F5EC;
  --color-forest-100: #C5E6CE;
  --color-forest-200: #9FD4AF;
  --color-forest-300: #79C290;
  --color-forest-400: #53B071;
  --color-forest-500: #2D9E52;
  --color-forest-600: #267F43;
  --color-forest-700: #1F6134;
  --color-forest-800: #1A4D2E;
  --color-forest-900: #0D2617;

  /* Brand: Gold (Accent) */
  --color-gold-50: #FFFBEB;
  --color-gold-100: #FEF3C7;
  --color-gold-200: #FDE68A;
  --color-gold-300: #FCD116;
  --color-gold-400: #FACC15;
  --color-gold-500: #EAB308;

  /* Brand: Clay (Secondary) */
  --color-clay-50: #F7F5F3;
  --color-clay-100: #EFEAE5;
  --color-clay-200: #DDD4C9;
  --color-clay-500: #A79275;
  --color-clay-600: #8B7355;
  --color-clay-700: #6B5842;

  /* Brand: Cream (Background) */
  --color-cream: #E8E4DF;

  /* Status colors (use Tailwind scale) */
  --color-red-50: #fef2f2;
  --color-red-500: #ef4444;
  --color-red-600: #dc2626;
  --color-red-700: #b91c1c;

  --color-green-50: #f0fdf4;
  --color-green-500: #22c55e;
  --color-green-600: #16a34a;
  --color-green-700: #15803d;

  --color-amber-50: #fffbeb;
  --color-amber-400: #fbbf24;
  --color-amber-500: #f59e0b;

  --color-blue-50: #eff6ff;
  --color-blue-500: #3b82f6;
  --color-blue-600: #2563eb;

  /* ========================================
     SPACING SCALE
     ======================================== */

  --spacing-px: 1px;
  --spacing-0: 0;
  --spacing-0-5: 0.125rem;  /* 2px */
  --spacing-1: 0.25rem;     /* 4px */
  --spacing-1-5: 0.375rem;  /* 6px */
  --spacing-2: 0.5rem;      /* 8px */
  --spacing-2-5: 0.625rem;  /* 10px */
  --spacing-3: 0.75rem;     /* 12px */
  --spacing-3-5: 0.875rem;  /* 14px */
  --spacing-4: 1rem;        /* 16px */
  --spacing-5: 1.25rem;     /* 20px */
  --spacing-6: 1.5rem;      /* 24px */
  --spacing-7: 1.75rem;     /* 28px */
  --spacing-8: 2rem;        /* 32px */
  --spacing-9: 2.25rem;     /* 36px */
  --spacing-10: 2.5rem;     /* 40px */
  --spacing-11: 2.75rem;    /* 44px */
  --spacing-12: 3rem;       /* 48px */
  --spacing-14: 3.5rem;     /* 56px */
  --spacing-16: 4rem;       /* 64px */
  --spacing-20: 5rem;       /* 80px */
  --spacing-24: 6rem;       /* 96px */

  /* ========================================
     TYPOGRAPHY
     ======================================== */

  --font-family-sans: 'Inter', system-ui, -apple-system, sans-serif;
  --font-family-mono: 'JetBrains Mono', ui-monospace, monospace;

  /* Font sizes with line heights */
  --text-xs: 0.75rem;       /* 12px */
  --text-xs-leading: 1rem;  /* 16px */

  --text-sm: 0.875rem;      /* 14px */
  --text-sm-leading: 1.25rem; /* 20px */

  --text-base: 1rem;        /* 16px */
  --text-base-leading: 1.5rem; /* 24px */

  --text-lg: 1.125rem;      /* 18px */
  --text-lg-leading: 1.75rem; /* 28px */

  --text-xl: 1.25rem;       /* 20px */
  --text-xl-leading: 1.75rem;

  --text-2xl: 1.5rem;       /* 24px */
  --text-2xl-leading: 2rem;

  --text-3xl: 1.875rem;     /* 30px */
  --text-3xl-leading: 2.25rem;

  /* Font weights */
  --font-weight-normal: 400;
  --font-weight-medium: 500;
  --font-weight-semibold: 600;
  --font-weight-bold: 700;

  /* ========================================
     BORDER RADIUS
     ======================================== */

  --radius-none: 0;
  --radius-sm: 0.125rem;    /* 2px */
  --radius-DEFAULT: 0.25rem; /* 4px */
  --radius-md: 0.375rem;    /* 6px */
  --radius-lg: 0.5rem;      /* 8px */
  --radius-xl: 0.75rem;     /* 12px */
  --radius-2xl: 1rem;       /* 16px */
  --radius-3xl: 1.5rem;     /* 24px */
  --radius-full: 9999px;

  /* ========================================
     SHADOWS
     ======================================== */

  --shadow-xs: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
  --shadow-sm: 0 1px 3px 0 rgba(0, 0, 0, 0.1), 0 1px 2px -1px rgba(0, 0, 0, 0.1);
  --shadow-DEFAULT: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -2px rgba(0, 0, 0, 0.1);
  --shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -2px rgba(0, 0, 0, 0.1);
  --shadow-lg: 0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -4px rgba(0, 0, 0, 0.1);
  --shadow-xl: 0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 8px 10px -6px rgba(0, 0, 0, 0.1);
  --shadow-2xl: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
  --shadow-inner: inset 0 2px 4px 0 rgba(0, 0, 0, 0.05);

  /* ========================================
     TRANSITIONS
     ======================================== */

  --duration-75: 75ms;
  --duration-100: 100ms;
  --duration-150: 150ms;
  --duration-200: 200ms;
  --duration-300: 300ms;
  --duration-500: 500ms;

  --ease-in: cubic-bezier(0.4, 0, 1, 1);
  --ease-out: cubic-bezier(0, 0, 0.2, 1);
  --ease-in-out: cubic-bezier(0.4, 0, 0.2, 1);

  /* ========================================
     Z-INDEX SCALE
     ======================================== */

  --z-dropdown: 50;
  --z-sticky: 100;
  --z-fixed: 200;
  --z-modal-backdrop: 300;
  --z-modal: 400;
  --z-popover: 500;
  --z-tooltip: 600;
}
```

### 2.2 Dark Mode Support

```css
/* Dark mode overrides - applied when .dark class is on <html> */
.dark {
  --color-background: var(--color-zinc-950);
  --color-surface: var(--color-zinc-900);
  --color-surface-raised: var(--color-zinc-800);
  --color-surface-overlay: rgba(39, 39, 42, 0.75);

  --color-text-primary: white;
  --color-text-secondary: var(--color-zinc-400);
  --color-text-tertiary: var(--color-zinc-500);

  --color-border: rgba(255, 255, 255, 0.1);
  --color-border-hover: rgba(255, 255, 255, 0.2);

  /* Adjust primary for dark mode if needed */
  --color-primary-subtle: rgba(26, 77, 46, 0.2);
  --color-danger-subtle: rgba(220, 38, 38, 0.2);
  --color-success-subtle: rgba(22, 163, 74, 0.2);
}
```

### 2.3 Theme Switching

```javascript
// web/static/js/theme.js
document.addEventListener('alpine:init', () => {
  Alpine.store('theme', {
    mode: localStorage.getItem('theme') || 'system',

    init() {
      this.apply();
      window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
        if (this.mode === 'system') this.apply();
      });
    },

    apply() {
      const isDark = this.mode === 'dark' ||
        (this.mode === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches);
      document.documentElement.classList.toggle('dark', isDark);
    },

    toggle() {
      this.mode = this.mode === 'dark' ? 'light' : 'dark';
      localStorage.setItem('theme', this.mode);
      this.apply();
    },

    set(mode) {
      this.mode = mode;
      localStorage.setItem('theme', mode);
      this.apply();
    }
  });
});
```

### 2.4 Creating a New Theme

To restyle for a different project, create a new theme file:

```css
/* web/static/css/themes/healthcare.css */
:root {
  /* Override semantic colors only */
  --color-primary: #0891b2;        /* cyan-600 */
  --color-primary-hover: #06b6d4;  /* cyan-500 */
  --color-primary-subtle: #ecfeff; /* cyan-50 */

  --color-accent: #f97316;         /* orange-500 */
  --color-accent-hover: #fb923c;   /* orange-400 */
  --color-accent-text: white;

  --color-background: #f8fafc;     /* slate-50 */

  /* All components automatically use new colors */
}
```

---

## 3. Alpine.js Patterns

### 3.1 Standard Interactive Patterns

#### Dropdown Menu
```javascript
// Replaces Headless UI Menu
Alpine.data('dropdown', () => ({
  open: false,
  activeIndex: -1,
  items: [],

  init() {
    this.items = [...this.$el.querySelectorAll('[role="menuitem"]')];
  },

  toggle() {
    this.open ? this.close() : this.openMenu();
  },

  openMenu() {
    this.open = true;
    this.activeIndex = -1;
    this.$nextTick(() => this.$refs.menu?.focus());
  },

  close() {
    this.open = false;
    this.activeIndex = -1;
    this.$refs.button?.focus();
  },

  onKeydown(e) {
    switch(e.key) {
      case 'ArrowDown':
        e.preventDefault();
        this.activeIndex = Math.min(this.activeIndex + 1, this.items.length - 1);
        this.items[this.activeIndex]?.focus();
        break;
      case 'ArrowUp':
        e.preventDefault();
        this.activeIndex = Math.max(this.activeIndex - 1, 0);
        this.items[this.activeIndex]?.focus();
        break;
      case 'Escape':
        this.close();
        break;
      case 'Tab':
        this.close();
        break;
    }
  },

  onItemKeydown(e, index) {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      this.items[index]?.click();
    }
  }
}));
```

#### Dialog/Modal
```javascript
// Replaces Headless UI Dialog
Alpine.data('dialog', (initialOpen = false) => ({
  open: initialOpen,

  show() {
    this.open = true;
    document.body.classList.add('overflow-hidden');
    this.$nextTick(() => {
      this.$refs.panel?.focus();
      this.trapFocus();
    });
  },

  close() {
    this.open = false;
    document.body.classList.remove('overflow-hidden');
  },

  trapFocus() {
    const focusableElements = this.$refs.panel?.querySelectorAll(
      'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
    );
    if (!focusableElements?.length) return;

    const first = focusableElements[0];
    const last = focusableElements[focusableElements.length - 1];

    this.$refs.panel?.addEventListener('keydown', (e) => {
      if (e.key !== 'Tab') return;

      if (e.shiftKey && document.activeElement === first) {
        e.preventDefault();
        last.focus();
      } else if (!e.shiftKey && document.activeElement === last) {
        e.preventDefault();
        first.focus();
      }
    });
  }
}));
```

#### Switch/Toggle
```javascript
// Replaces Headless UI Switch
Alpine.data('switchToggle', (initialValue = false) => ({
  enabled: initialValue,

  toggle() {
    this.enabled = !this.enabled;
    this.$dispatch('change', this.enabled);
  },

  onKeydown(e) {
    if (e.key === ' ' || e.key === 'Enter') {
      e.preventDefault();
      this.toggle();
    }
  }
}));
```

#### Combobox (Autocomplete)
```javascript
// Replaces Headless UI Combobox
Alpine.data('combobox', (config = {}) => ({
  open: false,
  query: '',
  activeIndex: -1,
  selectedValue: config.initialValue || null,
  options: config.options || [],

  get filteredOptions() {
    if (!this.query) return this.options;
    return this.options.filter(opt =>
      opt.label.toLowerCase().includes(this.query.toLowerCase())
    );
  },

  select(option) {
    this.selectedValue = option.value;
    this.query = option.label;
    this.open = false;
    this.$dispatch('change', option);
  },

  onInput() {
    this.open = true;
    this.activeIndex = 0;
  },

  onKeydown(e) {
    switch(e.key) {
      case 'ArrowDown':
        e.preventDefault();
        this.open = true;
        this.activeIndex = Math.min(this.activeIndex + 1, this.filteredOptions.length - 1);
        break;
      case 'ArrowUp':
        e.preventDefault();
        this.activeIndex = Math.max(this.activeIndex - 1, 0);
        break;
      case 'Enter':
        e.preventDefault();
        if (this.filteredOptions[this.activeIndex]) {
          this.select(this.filteredOptions[this.activeIndex]);
        }
        break;
      case 'Escape':
        this.open = false;
        break;
    }
  }
}));
```

### 3.2 Accessibility Patterns

All interactive components must include:

1. **ARIA attributes**:
   ```html
   <!-- Dropdown trigger -->
   <button
     aria-haspopup="true"
     :aria-expanded="open"
     aria-controls="menu-id">

   <!-- Dropdown menu -->
   <div
     id="menu-id"
     role="menu"
     aria-orientation="vertical"
     aria-labelledby="button-id">
   ```

2. **Focus management**:
   - Return focus to trigger when closing
   - Trap focus within modals
   - Support arrow key navigation in lists

3. **Screen reader announcements** (using live regions when needed):
   ```html
   <div aria-live="polite" aria-atomic="true" class="sr-only" x-text="announcement"></div>
   ```

### 3.3 Transition Classes

Standard Alpine.js transition patterns matching Catalyst's animations:

```html
<!-- Fade -->
x-transition:enter="transition ease-out duration-100"
x-transition:enter-start="opacity-0"
x-transition:enter-end="opacity-100"
x-transition:leave="transition ease-in duration-75"
x-transition:leave-start="opacity-100"
x-transition:leave-end="opacity-0"

<!-- Scale + Fade (for dropdowns) -->
x-transition:enter="transition ease-out duration-100"
x-transition:enter-start="opacity-0 scale-95"
x-transition:enter-end="opacity-100 scale-100"
x-transition:leave="transition ease-in duration-75"
x-transition:leave-start="opacity-100 scale-100"
x-transition:leave-end="opacity-0 scale-95"

<!-- Slide (for sidebar) -->
x-transition:enter="transition ease-in-out duration-300 transform"
x-transition:enter-start="-translate-x-full"
x-transition:enter-end="translate-x-0"
x-transition:leave="transition ease-in-out duration-300 transform"
x-transition:leave-start="translate-x-0"
x-transition:leave-end="-translate-x-full"

<!-- Modal slide up (mobile) -->
x-transition:enter="transition ease-out duration-100"
x-transition:enter-start="opacity-0 translate-y-12"
x-transition:enter-end="opacity-100 translate-y-0"
```

---

## 4. Component Porting Strategy

### 4.1 Complete Component Inventory (27 Components)

| # | Component | Complexity | Alpine.js | Dependencies | Priority |
|---|-----------|------------|-----------|--------------|----------|
| 1 | Text | Low | No | None | P0 |
| 2 | Heading | Low | No | None | P0 |
| 3 | Code | Low | No | None | P2 |
| 4 | Strong | Low | No | None | P2 |
| 5 | Link | Low | No | None | P0 |
| 6 | Divider | Low | No | None | P1 |
| 7 | Badge | Low | No | None | P0 |
| 8 | BadgeButton | Low | No | Badge, Link | P2 |
| 9 | Avatar | Low | No | None | P0 |
| 10 | AvatarButton | Low | No | Avatar, Link | P1 |
| 11 | Button | Medium | No | Link | P0 |
| 12 | Fieldset/Field/Label | Medium | No | None | P0 |
| 13 | Input | Medium | No | None | P0 |
| 14 | InputGroup | Medium | No | Input | P1 |
| 15 | Textarea | Medium | No | None | P0 |
| 16 | Select | Medium | No | None | P0 |
| 17 | Checkbox | Medium | No | None | P0 |
| 18 | Radio | Medium | No | None | P0 |
| 19 | Switch | Medium | Yes | None | P1 |
| 20 | Dropdown | High | Yes | Button | P0 |
| 21 | Dialog | High | Yes | Button, Text | P0 |
| 22 | Alert (Modal) | High | Yes | Dialog | P1 |
| 23 | Listbox | High | Yes | None | P2 |
| 24 | Combobox | High | Yes | Input | P2 |
| 25 | Table | Medium | No | Link | P0 |
| 26 | DescriptionList | Low | No | None | P1 |
| 27 | Pagination | Medium | No | Button | P1 |
| 28 | Navbar | Medium | Yes (mobile) | Link, Avatar | P1 |
| 29 | Sidebar | Medium | No | Link, Avatar | P0 |
| 30 | SidebarLayout | High | Yes | Sidebar, Navbar | P0 |
| 31 | StackedLayout | High | Yes | Navbar | P1 |

### 4.2 Dependency Graph

```
Level 0 (No dependencies):
  Text, Heading, Code, Strong, Divider, Badge, Avatar, Link
  Fieldset (Field, Label, Description, ErrorMessage)
  Input, Textarea, Select, Checkbox, Radio

Level 1 (Depend on Level 0):
  Button (Link)
  InputGroup (Input)
  BadgeButton (Badge, Link)
  AvatarButton (Avatar, Link)
  Switch
  Table (Link)
  DescriptionList

Level 2 (Depend on Level 1):
  Dropdown (Button)
  Dialog (Button, Text)
  Pagination (Button)
  Navbar (Link, Avatar)
  Sidebar (Link, Avatar)

Level 3 (Depend on Level 2):
  Alert (Dialog)
  Listbox (Input patterns)
  Combobox (Input)
  SidebarLayout (Sidebar, Navbar)
  StackedLayout (Navbar)
```

### 4.3 Priority Groups

**P0 - Core Foundation (Week 1-2)**
Must have for any page to function:
- Text, Heading, Link
- Button
- Fieldset (Field, FieldGroup, Label, Description, ErrorMessage)
- Input, Textarea, Select, Checkbox, Radio
- Badge, Avatar
- Table
- Dropdown (for user menu)
- Dialog (for confirmations)
- Sidebar, SidebarLayout

**P1 - Enhanced UX (Week 3-4)**
Important for good UX but not blocking:
- Divider
- Switch
- InputGroup
- AvatarButton
- Alert (modal)
- Pagination
- DescriptionList
- Navbar, StackedLayout

**P2 - Advanced Features (Week 5+)**
Nice to have, can defer:
- Code, Strong
- BadgeButton
- Listbox
- Combobox

---

## 5. Integration with Existing Lukaut Architecture

### 5.1 Template Engine Integration

The existing `/workspaces/lukaut/web/templates/` structure will be extended:

```go
// internal/template/engine.go - Update to include component templates
func (e *Engine) LoadTemplates() error {
    // Existing patterns
    patterns := []string{
        "web/templates/layouts/*.html",
        "web/templates/pages/**/*.html",
        "web/templates/partials/*.html",
        // New component patterns
        "web/templates/components/primitives/*.html",
        "web/templates/components/forms/*.html",
        "web/templates/components/interactive/*.html",
        "web/templates/components/data/*.html",
        "web/templates/components/navigation/*.html",
    }
    // ...
}
```

### 5.2 CSS Build Process

Update `/workspaces/lukaut/web/static/css/input.css`:

```css
/* Import design tokens first */
@import './themes/tokens.css';

/* Then Tailwind layers */
@tailwind base;
@tailwind components;
@tailwind utilities;

/* Component-specific CSS (for complex styling not possible with utilities) */
@import './components/button.css';
@import './components/input.css';
@import './components/dropdown.css';
@import './components/dialog.css';

/* Existing custom styles */
@layer base {
  /* ... existing base styles ... */
}

@layer components {
  /* Component classes will go here */
}

@layer utilities {
  /* ... existing utilities ... */
}
```

### 5.3 Tailwind Configuration Update

Update `/workspaces/lukaut/tailwind.config.js`:

```javascript
/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./web/templates/**/*.html",
    "./web/static/js/**/*.js",
  ],
  darkMode: 'class', // Enable class-based dark mode
  theme: {
    extend: {
      colors: {
        // Keep existing brand colors for backward compatibility
        'forest': { /* ... */ },
        'gold': { /* ... */ },
        'clay': { /* ... */ },
        'cream': '#E8E4DF',

        // Add CSS variable references for themeable colors
        'primary': 'var(--color-primary)',
        'primary-hover': 'var(--color-primary-hover)',
        'secondary': 'var(--color-secondary)',
        'accent': 'var(--color-accent)',
        'danger': 'var(--color-danger)',
        'surface': 'var(--color-surface)',
      },
      fontFamily: {
        sans: ['var(--font-family-sans)', 'system-ui', 'sans-serif'],
        mono: ['var(--font-family-mono)', 'monospace'],
      },
      borderRadius: {
        'DEFAULT': 'var(--radius-DEFAULT)',
        'lg': 'var(--radius-lg)',
        'xl': 'var(--radius-xl)',
      },
      boxShadow: {
        'xs': 'var(--shadow-xs)',
        'sm': 'var(--shadow-sm)',
        'DEFAULT': 'var(--shadow-DEFAULT)',
        'lg': 'var(--shadow-lg)',
      },
    },
  },
  plugins: [
    require('@tailwindcss/forms'),
  ],
}
```

### 5.4 Alpine.js Component Registration

Create `/workspaces/lukaut/web/static/js/alpine-components.js`:

```javascript
// Import this before Alpine.js initializes
document.addEventListener('alpine:init', () => {
  // Register all reusable Alpine components
  Alpine.data('dropdown', () => ({ /* ... */ }));
  Alpine.data('dialog', (initialOpen = false) => ({ /* ... */ }));
  Alpine.data('switchToggle', (initialValue = false) => ({ /* ... */ }));
  Alpine.data('combobox', (config = {}) => ({ /* ... */ }));
  Alpine.data('tabs', (initialTab = 0) => ({ /* ... */ }));

  // Global stores
  Alpine.store('theme', { /* ... */ });
  Alpine.store('toast', { /* for flash messages */ });
});
```

Update layout templates to include:
```html
<script src="/static/js/alpine-components.js"></script>
<script defer src="https://unpkg.com/alpinejs@3.x.x/dist/cdn.min.js"></script>
```

### 5.5 htmx Integration Points

Components should support htmx attributes via template parameters:

```html
{{define "button"}}
<button type="{{or .type "button"}}"
        class="btn btn-{{or .variant "solid"}} btn-{{or .color "primary"}}"
        {{if .hxGet}}hx-get="{{.hxGet}}"{{end}}
        {{if .hxPost}}hx-post="{{.hxPost}}"{{end}}
        {{if .hxPut}}hx-put="{{.hxPut}}"{{end}}
        {{if .hxDelete}}hx-delete="{{.hxDelete}}"{{end}}
        {{if .hxTarget}}hx-target="{{.hxTarget}}"{{end}}
        {{if .hxSwap}}hx-swap="{{.hxSwap}}"{{end}}
        {{if .hxConfirm}}hx-confirm="{{.hxConfirm}}"{{end}}
        {{if .hxIndicator}}hx-indicator="{{.hxIndicator}}"{{end}}
        {{if .disabled}}disabled{{end}}>
  {{if .loading}}
    <span class="htmx-indicator">
      {{template "spinner" dict "size" "sm"}}
    </span>
  {{end}}
  {{.text}}
</button>
{{end}}
```

---

## 6. Implementation Phases

### Phase 1: Foundation (Week 1-2)
**Goal:** Establish theming system and core primitives

**Tasks:**
1. Set up CSS variables theming system
   - Create `/web/static/css/themes/tokens.css`
   - Update `input.css` to import tokens
   - Update `tailwind.config.js` for CSS variable support

2. Port primitive components (no Alpine.js):
   - `text.html` (Text, Strong)
   - `heading.html` (Heading, Subheading)
   - `link.html`
   - `badge.html`
   - `avatar.html`
   - `divider.html`

3. Port Button component:
   - Full variant support (solid, outline, plain)
   - All color options
   - Size variants
   - htmx attribute pass-through

**Deliverable:** Working theme system + basic components

### Phase 2: Forms (Week 2-3)
**Goal:** Complete form component library

**Tasks:**
1. Port Fieldset components:
   - `fieldset.html` containing: Fieldset, Legend, FieldGroup, Field, Label, Description, ErrorMessage

2. Port form inputs:
   - `input.html` (with all states: default, hover, focus, disabled, invalid)
   - `input-group.html` (input with icons)
   - `textarea.html`
   - `select.html`

3. Port selection controls:
   - `checkbox.html` (Checkbox, CheckboxGroup, CheckboxField)
   - `radio.html` (Radio, RadioGroup, RadioField)
   - `switch.html` (requires Alpine.js)

**Deliverable:** Complete form system matching Catalyst aesthetics

### Phase 3: Interactive Components (Week 3-4)
**Goal:** Add client-side interactivity

**Tasks:**
1. Set up Alpine.js component library:
   - Create `/web/static/js/alpine-components.js`
   - Define dropdown, dialog, combobox, tabs data components

2. Port Dropdown:
   - `dropdown.html` with all sub-components
   - Full keyboard navigation
   - ARIA compliance

3. Port Dialog:
   - `dialog.html` with Title, Description, Body, Actions
   - Focus trapping
   - Backdrop click-to-close

4. Port Alert (modal confirm):
   - `alert.html` extending Dialog

**Deliverable:** Full interactive component suite

### Phase 4: Data & Navigation (Week 4-5)
**Goal:** Complete UI kit with data display and navigation

**Tasks:**
1. Port Table:
   - `table.html` with all sub-components
   - Row hover states
   - Clickable row support
   - Striped/dense/grid variants

2. Port DescriptionList:
   - `description-list.html`

3. Port Pagination:
   - `pagination.html` with Next/Previous/Page/Gap

4. Port Navigation:
   - `navbar.html`
   - `sidebar.html`
   - `sidebar-layout.html`
   - `stacked-layout.html`

**Deliverable:** Complete UI kit ready for production

### Phase 5: Migration & Polish (Week 5-6)
**Goal:** Migrate existing pages and polish

**Tasks:**
1. Update existing layouts:
   - Migrate `app.html` to use new SidebarLayout
   - Migrate `auth.html` to use new form components
   - Update `public.html` as needed

2. Update existing pages:
   - Dashboard
   - Login/Register/Verify
   - (Future pages)

3. Create documentation:
   - Component usage examples
   - Theming guide
   - Alpine.js patterns reference

4. Testing & polish:
   - Cross-browser testing
   - Accessibility audit
   - Performance optimization

**Deliverable:** Fully migrated application with documented component library

### Phase 6: Advanced Components (Future)
**Goal:** Add advanced interactive components

**Tasks:**
1. Listbox (custom select)
2. Combobox (autocomplete)
3. Tabs
4. Disclosure/Accordion
5. Toast notifications

---

## 7. Example Component Implementation

### Button Component

`/workspaces/lukaut/web/templates/components/primitives/button.html`:

```html
{{/*
  Button Component

  Params:
    text      - Button label (required)
    variant   - solid (default), outline, plain
    color     - primary (default), secondary, danger, neutral (solid only)
    size      - sm, md (default), lg
    type      - button (default), submit, reset
    href      - If set, renders as anchor tag
    disabled  - Boolean
    class     - Additional CSS classes
    icon      - Icon name (prepended)
    iconRight - Icon name (appended)

    htmx attributes:
    hxGet, hxPost, hxPut, hxDelete, hxPatch
    hxTarget, hxSwap, hxTrigger, hxIndicator, hxConfirm

  Usage:
    {{template "button" dict "text" "Save Changes" "type" "submit" "color" "primary"}}
    {{template "button" dict "text" "Cancel" "variant" "outline" "href" "/back"}}
    {{template "button" dict "text" "Delete" "color" "danger" "hxDelete" "/api/item/1" "hxConfirm" "Are you sure?"}}
*/}}

{{define "button"}}
{{- $variant := or .variant "solid" -}}
{{- $color := or .color "primary" -}}
{{- $size := or .size "md" -}}
{{- $type := or .type "button" -}}

{{- /* Base classes */ -}}
{{- $base := "relative isolate inline-flex items-center justify-center gap-x-2 rounded-lg border font-semibold focus:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-colors" -}}

{{- /* Size classes */ -}}
{{- $sizeClasses := "" -}}
{{- if eq $size "sm" -}}
  {{- $sizeClasses = "px-2.5 py-1.5 text-sm" -}}
{{- else if eq $size "lg" -}}
  {{- $sizeClasses = "px-4 py-3 text-base" -}}
{{- else -}}
  {{- $sizeClasses = "px-3.5 py-2.5 text-sm sm:px-3 sm:py-1.5" -}}
{{- end -}}

{{- /* Variant + Color classes */ -}}
{{- $variantClasses := "" -}}
{{- if eq $variant "outline" -}}
  {{- $variantClasses = "border-[var(--color-border)] bg-transparent text-[var(--color-text-primary)] hover:bg-[var(--color-zinc-950-5)] focus-visible:ring-[var(--color-primary)]" -}}
{{- else if eq $variant "plain" -}}
  {{- $variantClasses = "border-transparent bg-transparent text-[var(--color-text-primary)] hover:bg-[var(--color-zinc-950-5)] focus-visible:ring-[var(--color-primary)]" -}}
{{- else -}}
  {{- /* Solid variant - color matters */ -}}
  {{- if eq $color "danger" -}}
    {{- $variantClasses = "border-transparent bg-[var(--color-danger)] text-white hover:bg-[var(--color-danger-hover)] focus-visible:ring-[var(--color-danger)]" -}}
  {{- else if eq $color "secondary" -}}
    {{- $variantClasses = "border-transparent bg-[var(--color-secondary)] text-white hover:bg-[var(--color-secondary-hover)] focus-visible:ring-[var(--color-secondary)]" -}}
  {{- else if eq $color "neutral" -}}
    {{- $variantClasses = "border-[var(--color-border)] bg-white text-[var(--color-text-primary)] hover:bg-[var(--color-zinc-50)] focus-visible:ring-[var(--color-primary)] shadow-sm" -}}
  {{- else -}}
    {{- /* Primary (default) */ -}}
    {{- $variantClasses = "border-transparent bg-[var(--color-primary)] text-[var(--color-text-on-primary)] hover:bg-[var(--color-primary-hover)] focus-visible:ring-[var(--color-primary)]" -}}
  {{- end -}}
{{- end -}}

{{- $allClasses := printf "%s %s %s %s" $base $sizeClasses $variantClasses (or .class "") -}}

{{if .href}}
<a href="{{.href}}"
   class="{{$allClasses}}"
   {{if .disabled}}aria-disabled="true" tabindex="-1"{{end}}>
  {{if .icon}}{{template "icon" dict "name" .icon "class" "size-5 sm:size-4 shrink-0"}}{{end}}
  <span>{{.text}}</span>
  {{if .iconRight}}{{template "icon" dict "name" .iconRight "class" "size-5 sm:size-4 shrink-0"}}{{end}}
</a>
{{else}}
<button type="{{$type}}"
        class="{{$allClasses}}"
        {{if .disabled}}disabled{{end}}
        {{if .hxGet}}hx-get="{{.hxGet}}"{{end}}
        {{if .hxPost}}hx-post="{{.hxPost}}"{{end}}
        {{if .hxPut}}hx-put="{{.hxPut}}"{{end}}
        {{if .hxDelete}}hx-delete="{{.hxDelete}}"{{end}}
        {{if .hxPatch}}hx-patch="{{.hxPatch}}"{{end}}
        {{if .hxTarget}}hx-target="{{.hxTarget}}"{{end}}
        {{if .hxSwap}}hx-swap="{{.hxSwap}}"{{end}}
        {{if .hxTrigger}}hx-trigger="{{.hxTrigger}}"{{end}}
        {{if .hxIndicator}}hx-indicator="{{.hxIndicator}}"{{end}}
        {{if .hxConfirm}}hx-confirm="{{.hxConfirm}}"{{end}}>
  {{if .icon}}{{template "icon" dict "name" .icon "class" "size-5 sm:size-4 shrink-0"}}{{end}}
  <span>{{.text}}</span>
  {{if .iconRight}}{{template "icon" dict "name" .iconRight "class" "size-5 sm:size-4 shrink-0"}}{{end}}
</button>
{{end}}
{{end}}
```

### Dropdown Component

`/workspaces/lukaut/web/templates/components/interactive/dropdown.html`:

```html
{{/*
  Dropdown Component

  Params:
    trigger     - Content for trigger button (template name or text)
    triggerText - Simple text for trigger (alternative to trigger)
    align       - start (default), end
    width       - auto (default), sm, md, lg
    items       - Slice of dropdown items (see structure below)

  Item structure:
    { label: string, href?: string, icon?: string, separator?: bool, disabled?: bool }

  Usage:
    {{template "dropdown" dict "triggerText" "Options" "items" .MenuItems "align" "end"}}
*/}}

{{define "dropdown"}}
<div x-data="dropdown" class="relative inline-block text-left">
  <!-- Trigger -->
  <button type="button"
          x-ref="button"
          @click="toggle()"
          @keydown.arrow-down.prevent="openMenu()"
          @keydown.arrow-up.prevent="openMenu()"
          aria-haspopup="true"
          :aria-expanded="open"
          class="{{or .triggerClass "inline-flex items-center gap-x-1.5 rounded-lg bg-white px-3 py-2 text-sm font-semibold text-[var(--color-text-primary)] shadow-sm ring-1 ring-inset ring-[var(--color-border)] hover:bg-[var(--color-zinc-50)]"}}">
    {{if .triggerText}}
      {{.triggerText}}
    {{else if .trigger}}
      {{template .trigger .}}
    {{end}}
    <svg class="size-5 text-[var(--color-text-secondary)]" viewBox="0 0 20 20" fill="currentColor">
      <path fill-rule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 11.168l3.71-3.938a.75.75 0 111.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z" clip-rule="evenodd" />
    </svg>
  </button>

  <!-- Menu -->
  <div x-ref="menu"
       x-show="open"
       x-cloak
       @click.away="close()"
       @keydown.escape.window="close()"
       @keydown="onKeydown($event)"
       x-transition:enter="transition ease-out duration-100"
       x-transition:enter-start="opacity-0 scale-95"
       x-transition:enter-end="opacity-100 scale-100"
       x-transition:leave="transition ease-in duration-75"
       x-transition:leave-start="opacity-100 scale-100"
       x-transition:leave-end="opacity-0 scale-95"
       class="absolute {{if eq .align "end"}}right-0{{else}}left-0{{end}} z-[var(--z-dropdown)] mt-2 {{if eq .width "sm"}}w-40{{else if eq .width "md"}}w-56{{else if eq .width "lg"}}w-72{{else}}w-auto min-w-[12rem]{{end}} origin-top-{{or .align "start"}} rounded-xl bg-[var(--color-surface-overlay)] p-1 shadow-lg ring-1 ring-[var(--color-border)] backdrop-blur-xl focus:outline-none"
       role="menu"
       aria-orientation="vertical"
       tabindex="-1">
    {{range $i, $item := .items}}
      {{if $item.separator}}
        <div class="my-1 h-px bg-[var(--color-border)]" role="separator"></div>
      {{else}}
        {{if $item.href}}
        <a href="{{$item.href}}"
           role="menuitem"
           tabindex="-1"
           @keydown="onItemKeydown($event, {{$i}})"
           class="group flex w-full items-center gap-x-3 rounded-lg px-3 py-2 text-sm text-[var(--color-text-primary)] hover:bg-[var(--color-primary)] hover:text-white focus:bg-[var(--color-primary)] focus:text-white focus:outline-none {{if $item.disabled}}opacity-50 pointer-events-none{{end}}">
          {{if $item.icon}}
            {{template "icon" dict "name" $item.icon "class" "size-5 text-[var(--color-text-secondary)] group-hover:text-white group-focus:text-white"}}
          {{end}}
          {{$item.label}}
        </a>
        {{else}}
        <button type="button"
                role="menuitem"
                tabindex="-1"
                @keydown="onItemKeydown($event, {{$i}})"
                {{if $item.onClick}}@click="{{$item.onClick}}"{{end}}
                class="group flex w-full items-center gap-x-3 rounded-lg px-3 py-2 text-sm text-[var(--color-text-primary)] hover:bg-[var(--color-primary)] hover:text-white focus:bg-[var(--color-primary)] focus:text-white focus:outline-none {{if $item.disabled}}opacity-50 pointer-events-none{{end}}">
          {{if $item.icon}}
            {{template "icon" dict "name" $item.icon "class" "size-5 text-[var(--color-text-secondary)] group-hover:text-white group-focus:text-white"}}
          {{end}}
          {{$item.label}}
        </button>
        {{end}}
      {{end}}
    {{end}}
  </div>
</div>
{{end}}
```

---

## 8. Decisions (Resolved)

The following architectural decisions have been finalized:

1. **Motion/Animations**: ✅ **Skip animations**
   - No Framer Motion replacement needed
   - Sidebar indicator will use static styling (CSS class for current item)
   - Can revisit with CSS transitions later if desired

2. **Icon strategy**: ✅ **SVG inline via Go template partial**
   - Create `{{template "icon" dict "name" "chevron-down" "class" "size-5"}}`
   - Icons rendered as inline SVG for flexibility and no extra HTTP requests
   - Use Heroicons icon set (matches Catalyst)

3. **Form validation**: ✅ **htmx-based inline validation**
   - Server validates on blur/change events
   - Returns error HTML partials that replace field error messages
   - Single source of truth (server), good UX (no page reload)
   - Components support `invalid` parameter for error styling

4. **Dark mode**: ✅ **Foundation in Phase 1, full implementation Phase 5+**
   - CSS variable structure supports dark mode from day one
   - `.dark` class variables defined but not all components styled
   - Full dark mode styling deferred until core components complete

5. **Testing strategy**: Manual testing checklist for MVP
   - Visual regression testing can be added in Phase 6
   - Focus on keyboard navigation and ARIA compliance during development

---

## 9. Success Criteria

The port is complete when:

1. [ ] All P0 and P1 components are ported and documented
2. [ ] Existing Lukaut pages use the new component system
3. [ ] Theme switching works (at minimum light mode with CSS variables)
4. [ ] All interactive components pass keyboard navigation tests
5. [ ] All components have proper ARIA attributes
6. [ ] A new theme can be created by overriding ~20 CSS variables
7. [ ] Component documentation exists with usage examples

---

## Appendix A: Catalyst to Template Mapping

| Catalyst Component | Go Template Name | Notes |
|--------------------|------------------|-------|
| `<Text>` | `{{template "text" .}}` | |
| `<Strong>` | `{{template "strong" .}}` | |
| `<Code>` | `{{template "code" .}}` | |
| `<Heading>` | `{{template "heading" .}}` | |
| `<Subheading>` | `{{template "subheading" .}}` | |
| `<Link>` | `{{template "link" .}}` | |
| `<Button>` | `{{template "button" .}}` | |
| `<Badge>` | `{{template "badge" .}}` | |
| `<Avatar>` | `{{template "avatar" .}}` | |
| `<Divider>` | `{{template "divider" .}}` | |
| `<Fieldset>` | `{{template "fieldset" .}}` | |
| `<Legend>` | `{{template "legend" .}}` | |
| `<FieldGroup>` | `{{template "field-group" .}}` | |
| `<Field>` | `{{template "field" .}}` | |
| `<Label>` | `{{template "label" .}}` | |
| `<Description>` | `{{template "field-description" .}}` | |
| `<ErrorMessage>` | `{{template "field-error" .}}` | |
| `<Input>` | `{{template "input" .}}` | |
| `<InputGroup>` | `{{template "input-group" .}}` | |
| `<Textarea>` | `{{template "textarea" .}}` | |
| `<Select>` | `{{template "select" .}}` | |
| `<Checkbox>` | `{{template "checkbox" .}}` | |
| `<Radio>` | `{{template "radio" .}}` | |
| `<Switch>` | `{{template "switch" .}}` | |
| `<Dropdown>` | `{{template "dropdown" .}}` | |
| `<Dialog>` | `{{template "dialog" .}}` | |
| `<Alert>` (modal) | `{{template "alert-dialog" .}}` | |
| `<Listbox>` | `{{template "listbox" .}}` | |
| `<Combobox>` | `{{template "combobox" .}}` | |
| `<Table>` | `{{template "table" .}}` | |
| `<DescriptionList>` | `{{template "description-list" .}}` | |
| `<Pagination>` | `{{template "pagination" .}}` | |
| `<Navbar>` | `{{template "navbar" .}}` | |
| `<Sidebar>` | `{{template "sidebar" .}}` | |
| `<SidebarLayout>` | `{{template "sidebar-layout" .}}` | |
| `<StackedLayout>` | `{{template "stacked-layout" .}}` | |

---

## Appendix B: Headless UI to Alpine.js Mapping

| Headless UI | Alpine.js Equivalent |
|-------------|---------------------|
| `<Menu>` | `x-data="dropdown"` |
| `<MenuButton>` | `@click="toggle()"` with aria attributes |
| `<MenuItems>` | `x-show="open"` with transitions |
| `<MenuItem>` | Role="menuitem" with focus handling |
| `<Dialog>` | `x-data="dialog"` |
| `<DialogBackdrop>` | `x-show="open"` fixed overlay |
| `<DialogPanel>` | Focus trap with `x-ref` |
| `<Switch>` | `x-data="switchToggle"` |
| `<Listbox>` | `x-data="listbox"` |
| `<Combobox>` | `x-data="combobox"` |
| `<RadioGroup>` | Native radio inputs with styling |
| `<Disclosure>` | `x-data="{ open: false }"` simple toggle |
| `<Tab>` | `x-data="tabs"` |
| `transition` prop | Alpine `x-transition` directives |
| `data-*` attributes | `:class` bindings or `x-bind` |
