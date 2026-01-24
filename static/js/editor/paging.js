/**
 * Template Editor - Paging Module
 * Handles manual page breaks and page count tracking
 */

function createPagingBehavior() {
    return {
        /**
         * Insert a manual page break at cursor position
         */
        insertPageBreak() {
            const pageBreak = '<div class="page-break" contenteditable="false" style="page-break-after: always; user-select: none; margin: 0; position: relative; border-top: 1px dashed #ccc; width: 100%;"><span style="position: absolute; top: -10px; right: 0; background: #eee; padding: 2px 6px; font-size: 10px; color: #666; border-radius: 4px; pointer-events: none;">Page Break</span></div><p><br></p>';
            document.execCommand('insertHTML', false, pageBreak);
            
            this.$nextTick(() => {
                this.updatePageCount();
                this.updatePageBreaks();
            });
        },

        /**
         * Update the visual positioning of page breaks
         * Uses batched read/write operations to avoid layout thrashing
         */
        updatePageBreaks() {
            const editor = document.getElementById('editor-content');
            if (!editor) return;

            const breaks = Array.from(editor.querySelectorAll('.page-break'));
            const totalPageHeight = this.pageHeight + this.pageGap;

            // BATCH 1: Reset all heights (WRITE)
            breaks.forEach(el => el.style.height = '0px');

            // BATCH 2: Measure all (READ)
            const editorRect = editor.getBoundingClientRect();
            const editorScrollTop = editor.scrollTop;
            
            const measurements = breaks.map(el => {
                const rect = el.getBoundingClientRect();
                const relativeTop = rect.top - editorRect.top + editorScrollTop;
                return { el, relativeTop };
            });

            // BATCH 3: Calculate and Apply (WRITE)
            let currentShift = 0;

            measurements.forEach(({ el, relativeTop }) => {
                const pos = relativeTop + currentShift;
                const pageIndex = Math.floor(pos / totalPageHeight);
                const nextPageContentStart = (pageIndex + 1) * totalPageHeight + this.pagePadding;
                
                let neededHeight = nextPageContentStart - pos;
                if (neededHeight < 0) neededHeight = 0;

                currentShift += neededHeight;
                el.style.height = neededHeight + 'px';
            });
        },

        /**
         * Handle keyboard shortcuts
         */
        handleKeydown(event) {
            if (event.ctrlKey && event.key === 'Enter') {
                event.preventDefault();
                this.insertPageBreak();
            }
            // Debounce/Throttle the update
            if (this.pagingRaf) cancelAnimationFrame(this.pagingRaf);
            this.pagingRaf = requestAnimationFrame(() => this.updatePageBreaks());
        },

        /**
         * Update total page count based on content height
         */
        updatePageCount() {
            const editor = document.getElementById('editor-content');
            if (!editor) return;
            
            const contentHeight = editor.scrollHeight;
            const effectivePageHeight = this.pageHeight + this.pageGap;
            this.totalPages = Math.max(1, Math.ceil(contentHeight / effectivePageHeight));
        },

        /**
         * Update current page indicator based on scroll position
         */
        updateCurrentPage() {
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
