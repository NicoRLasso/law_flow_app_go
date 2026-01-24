/**
 * Template Editor - Config Module
 * Basic editor configuration settings
 */

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
