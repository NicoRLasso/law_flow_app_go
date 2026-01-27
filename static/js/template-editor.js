/**
 * Template Editor - Main Entry Point
 * Composes all behavior modules into a single Alpine.js data object
 */

function templateEditor(configuredPageHeight = 1056) {
    return {
        // Compose behaviors from separate modules
        ...getEditorConfig(configuredPageHeight),
        ...createSelectionBehavior(),
        ...createMenuBehavior(),
        ...createFormattingBehavior(),
        ...createPagingBehavior(),
        ...createAutoPagingBehavior(),
        ...createZoomBehavior(),

        // Shared state
        debounceTimer: null,
        variables: [],
        openDropdown: '', // Track which dropdown is open (heading, fontSize, color)
        showColorAdvanced: false, // Toggle for advanced color picker

        init() {
            // Load variables
            this.loadVariables();

            // Listen for selection changes within the editor
            document.addEventListener('selectionchange', () => {
                this.updateFormattingState();
                this.trackEditorSelection();
            });

            // Initial sizing
            this.$nextTick(() => {
                this.updatePageCount();
                this.updatePageBreaks();
                // Run initial auto-paging check to render breaks for loaded content
                this.checkAutoPageBreaks();
            });
        },

        destroy() {
            // Clean up timers to prevent memory leaks
            clearTimeout(this.debounceTimer);
            if (this.inputRaf) cancelAnimationFrame(this.inputRaf);
            if (this.pagingRaf) cancelAnimationFrame(this.pagingRaf);
            if (this.scrollRaf) cancelAnimationFrame(this.scrollRaf);
        },

        // Combined Input Handler
        handleInput() {
            // Update visual page breaks immediately but efficiently
            if (this.inputRaf) cancelAnimationFrame(this.inputRaf);
            this.inputRaf = requestAnimationFrame(() => {
                this.updatePageCount();
                this.updatePageBreaks();
            });

            // Debounce auto-paging (heavy calculation)
            clearTimeout(this.debounceTimer);
            this.debounceTimer = setTimeout(() => {
                this.checkAutoPageBreaks();
            }, 1000);
        },

        async loadVariables() {
            try {
                const response = await fetch('/api/templates/variables');
                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }
                this.variables = await response.json();
            } catch (e) {
                console.error('Failed to load variables:', e);
                this.variables = [];
            }
        }
    };
}

/**
 * Save template content - Global function for form submission
 */
function saveTemplateContent() {
    const editor = document.getElementById('editor-content');
    if (!editor) return;

    // Clone to clean
    const clone = editor.cloneNode(true);
    
    // Remove auto-breaks
    const autoBreaks = clone.querySelectorAll('.auto-break');
    autoBreaks.forEach(el => el.remove());

    const content = clone.innerHTML;
    document.getElementById('hidden-content-input').value = content;
    htmx.trigger('#save-template-form', 'submit');
}
