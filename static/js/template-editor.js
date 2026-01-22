function templateEditor(configuredPageHeight = 1056) {
    return {
        // Compose behaviors
        ...getEditorConfig(configuredPageHeight),
        ...createMenuBehavior(),
        ...createFormattingBehavior(),
        ...createPagingBehavior(),
        ...createAutoPagingBehavior(),

        // debounceTimer matches Config but is stateful, so typically put here or in behaviors. 
        // We'll let createAutoPagingBehavior handle its own timer if needed, or keep shared state here.
        debounceTimer: null,
        variables: [],

        init() {
            // Load variables
            this.loadVariables();

            // Listen for selection changes
            document.addEventListener('selectionchange', () => this.updateFormattingState());

            // Initial sizing
            this.$nextTick(() => {
                this.updatePageCount();
                this.updatePageBreaks();
            });
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
                this.variables = await response.json();
            } catch (e) {
                console.error('Failed to load variables:', e);
            }
        }
    };
}

// --- Behavior Modules ---

function getEditorConfig(pageHeight) {
    return {
        viewMode: 'paginated',
        pageHeight: pageHeight,
        pageWidth: pageHeight === 1123 ? 794 : 816, // A4 vs Letter
        pageGap: 32,
        pagePadding: 96,
        currentPage: 1,
        totalPages: 1
    };
}

function createMenuBehavior() {
    return {
        showContextMenu: false,
        contextMenuX: 0,
        contextMenuY: 0,

        getContextMenuStyle() {
            return `top: ${this.contextMenuY}px; left: ${this.contextMenuX}px`;
        },

        openContextMenu(event) {
            const menuWidth = 256;
            const menuHeight = 320;
            const padding = 10;
            
            let x = event.clientX;
            let y = event.clientY;

            if (x + menuWidth + padding > window.innerWidth) x = window.innerWidth - menuWidth - padding;
            if (y + menuHeight + padding > window.innerHeight) y = window.innerHeight - menuHeight - padding;

            this.contextMenuX = Math.max(padding, x);
            this.contextMenuY = Math.max(padding, y);
            this.showContextMenu = true;
        },

        insertVariable(tag) {
            this.insertTextAtCursor("{{ " + tag + " }}");
            this.showContextMenu = false;
        },

        insertTextAtCursor(text) {
            const sel = window.getSelection();
            if (sel.rangeCount) {
                const range = sel.getRangeAt(0);
                range.deleteContents();
                const textNode = document.createTextNode(text);
                range.insertNode(textNode);
                
                range.setStartAfter(textNode);
                range.setEndAfter(textNode);
                sel.removeAllRanges();
                sel.addRange(range);
                
                const editor = document.getElementById('editor-content');
                if (editor) editor.dispatchEvent(new Event('input', { bubbles: true }));
            }
        }
    };
}

function createFormattingBehavior() {
    return {
        isBold: false,
        isItalic: false,
        isUnderline: false,
        alignment: 'left',

        execCommand(command) {
            document.execCommand(command, false, null);
            this.updateFormattingState();
        },

        updateFormattingState() {
            this.isBold = document.queryCommandState('bold');
            this.isItalic = document.queryCommandState('italic');
            this.isUnderline = document.queryCommandState('underline');

            if (document.queryCommandState('justifyLeft')) this.alignment = 'left';
            else if (document.queryCommandState('justifyCenter')) this.alignment = 'center';
            else if (document.queryCommandState('justifyRight')) this.alignment = 'right';
            else if (document.queryCommandState('justifyFull')) this.alignment = 'justify';
        }
    };
}

function createPagingBehavior() {
    return {
        insertPageBreak() {
            const pageBreak = '<div class="page-break" contenteditable="false" style="page-break-after: always; user-select: none; margin: 0; position: relative; border-top: 1px dashed #ccc; width: 100%;"><span style="position: absolute; top: -10px; right: 0; background: #eee; padding: 2px 6px; font-size: 10px; color: #666; border-radius: 4px; pointer-events: none;">Page Break</span></div><p><br></p>';
            document.execCommand('insertHTML', false, pageBreak);
            
            this.$nextTick(() => {
                this.updatePageCount();
                this.updatePageBreaks();
            });
        },

        updatePageBreaks() {
            const editor = document.getElementById('editor-content');
            if (!editor) return;

            const breaks = Array.from(editor.querySelectorAll('.page-break'));
            const totalPageHeight = this.pageHeight + this.pageGap;

            // BATCH 1: Reset all heights (WRITE)
            // This is necessary to measure the "natural" position of elements without the spacers.
            breaks.forEach(el => el.style.height = '0px');

            // BATCH 2: Measure all (READ)
            // This triggers ONE reflow for the whole set
            const editorRect = editor.getBoundingClientRect();
            const editorScrollTop = editor.scrollTop;
            
            const measurements = breaks.map(el => {
                const rect = el.getBoundingClientRect();
                // Calculate position relative to the top of the scrolled content
                const relativeTop = rect.top - editorRect.top + editorScrollTop;
                return { el, relativeTop };
            });

            // BATCH 3: Calculate and Apply (WRITE)
            // We track the cumulative shift caused by previous spacers to adjust downstream calculations
            let currentShift = 0;

            measurements.forEach(({ el, relativeTop }) => {
                // The effective position is the natural position + shift from previous spacers
                const pos = relativeTop + currentShift;
                
                const pageIndex = Math.floor(pos / totalPageHeight);
                const nextPageContentStart = (pageIndex + 1) * totalPageHeight + this.pagePadding;
                
                let neededHeight = nextPageContentStart - pos;
                if (neededHeight < 0) neededHeight = 0;

                // Update shift for subsequent elements
                currentShift += neededHeight;

                el.style.height = neededHeight + 'px';
            });
        },

        handleKeydown(event) {
            if (event.ctrlKey && event.key === 'Enter') {
                event.preventDefault();
                this.insertPageBreak();
            }
            // Debounce/Throttle the update
            if (this.pagingRaf) cancelAnimationFrame(this.pagingRaf);
            this.pagingRaf = requestAnimationFrame(() => this.updatePageBreaks());
        },

        updatePageCount() {
            const editor = document.getElementById('editor-content');
            if (!editor) return;
            // scrollHeight causes reflow, so it fits in the measuring phase ideally, 
            // but for now keeping it simple.
            const contentHeight = editor.scrollHeight;
            const effectivePageHeight = this.pageHeight + this.pageGap;
            this.totalPages = Math.max(1, Math.ceil(contentHeight / effectivePageHeight));
        },

        updateCurrentPage() {
            // Throttling this could also be good, but scroll events fire rapidly
            if (this.scrollRaf) return;
            this.scrollRaf = requestAnimationFrame(() => {
                const scroller = document.getElementById('editor-scroller');
                if (!scroller) {
                    this.scrollRaf = null;
                    return;
                }
                const scrollTop = scroller.scrollTop;
                const effectivePageHeight = this.pageHeight + this.pageGap;
                const middle = scrollTop + (scroller.clientHeight / 3);
                this.currentPage = Math.max(1, Math.min(
                    this.totalPages,
                    Math.floor(middle / effectivePageHeight) + 1
                ));
                this.scrollRaf = null;
            });
        }
    };
}

function createAutoPagingBehavior() {
    return {
        checkAutoPageBreaks() {
            const editor = document.getElementById('editor-content');
            if (!editor) return;
            
            const selection = this.saveSelection();
            
            // PHASE 1: Reset
            const existingAutoBreaks = editor.querySelectorAll('.auto-break');
            existingAutoBreaks.forEach(el => el.remove());
            
            // PHASE 2: Measure
            const children = Array.from(editor.children);
            const totalPageHeight = this.pageHeight + this.pageGap;
            
            // Optimization: Use offsetTop/offsetHeight instead of getBoundingClientRect
            // checks to avoid layout thrashing per element.
            // Since editor-content is 'relative', offsetTop is relative to it.
            
            const measurements = children.map(child => {
                if (child.classList.contains('page-break')) return null;
                if (child.style.display === 'none') return null;
                
                return {
                    el: child,
                    top: child.offsetTop,
                    height: child.offsetHeight,
                    prevIsManualBreak: child.previousElementSibling && 
                                     child.previousElementSibling.classList.contains('page-break') && 
                                     !child.previousElementSibling.classList.contains('auto-break')
                };
            }).filter(item => item !== null);

            // PHASE 3: Calculate
            const insertions = [];
            for (const item of measurements) {
                if (item.height > this.pageHeight * 0.8) continue;

                const pageIndex = Math.floor(item.top / totalPageHeight);
                const pageStart = pageIndex * totalPageHeight;
                const pageContentLimit = pageStart + this.pageHeight - this.pagePadding;

                if (item.top < pageContentLimit && (item.top + item.height) > pageContentLimit) {
                     if (item.prevIsManualBreak) continue;
                     insertions.push(item.el);
                }
            }

            // PHASE 4: Apply
            if (insertions.length > 0) {
                 insertions.forEach(targetEl => {
                     const br = document.createElement('div');
                     br.className = 'page-break auto-break';
                     br.contentEditable = "false";
                     br.style.cssText = 'display: block; width: 100%; border-top: 1px dotted #ccc; margin: 0; opacity: 0.5;';
                     br.innerHTML = '<span style="position: absolute; right: 0; font-size: 8px; color: #999; pointer-events: none; padding-right: 4px;">Auto</span>';
                     editor.insertBefore(br, targetEl);
                 });

                 this.updatePageBreaks();
                 this.updatePageCount();
                 this.restoreSelection(selection);
            } else if (existingAutoBreaks.length > 0) {
                this.updatePageBreaks();
                this.updatePageCount();
                this.restoreSelection(selection);
            }
        },

        saveSelection() {
            const sel = window.getSelection();
            if (!sel.rangeCount) return null;
            const range = sel.getRangeAt(0);
            return {
                startContainer: range.startContainer,
                startOffset: range.startOffset,
                endContainer: range.endContainer,
                endOffset: range.endOffset
            };
        },

        restoreSelection(saved) {
            if (!saved) return;
            
            // Safety: check if nodes remain connected
            if (!document.contains(saved.startContainer) || !document.contains(saved.endContainer)) {
                return;
            }

            try {
                const range = document.createRange();
                range.setStart(saved.startContainer, saved.startOffset);
                range.setEnd(saved.endContainer, saved.endOffset);
                
                // Only if range creation succeeded do we update the DOM selection
                const sel = window.getSelection();
                sel.removeAllRanges();
                sel.addRange(range);
            } catch (e) {
                console.warn('Could not restore selection safely:', e);
            }
        }
    };
}

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
