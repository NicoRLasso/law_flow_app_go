console.log('Law Flow App - Custom JavaScript loaded');

// HTMX event listeners for debugging and custom behavior
document.addEventListener('htmx:beforeRequest', function(event) {
    console.log('HTMX Request starting:', event.detail);
});

document.addEventListener('htmx:afterRequest', function(event) {
    console.log('HTMX Request completed:', event.detail);
});

document.addEventListener('htmx:responseError', function(event) {
    console.error('HTMX Error:', event.detail);
});

// Alpine.js global store (if needed)
document.addEventListener('alpine:init', () => {
    console.log('Alpine.js initialized');
});
