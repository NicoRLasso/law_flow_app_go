/* ============================================
   KINETIC TYPOGRAPHY - SCROLL ANIMATIONS
   ============================================ */

// Check for reduced motion preference globally
const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

// Hero Parallax Effect
function initHeroParallax() {
    const hero = document.querySelector('.hero-parallax');
    if (!hero) return;

    if (prefersReducedMotion) return;

    let heroHeight = hero.offsetHeight;
    let ticking = false;

    window.addEventListener('resize', () => {
        heroHeight = hero.offsetHeight;
    });

    window.addEventListener('scroll', () => {
        if (!ticking) {
            window.requestAnimationFrame(() => {
                const scrolled = window.window.pageYOffset || document.documentElement.scrollTop;
                const scrollPercent = Math.min(1, Math.max(0, scrolled / heroHeight));

                // Scale from 1.0 to 1.2 and fade out
                const scale = 1 + (scrollPercent * 0.2);
                const opacity = Math.max(0, 1 - (scrollPercent * 2));

                hero.style.transform = `scale(${scale})`;
                hero.style.opacity = opacity;
                
                ticking = false;
            });
            ticking = true;
        }
    }, { passive: true });
}

// Intersection Observer for fade-in animations
function initFadeInObserver() {
    const fadeElements = document.querySelectorAll('.fade-in');
    if (fadeElements.length === 0) return;

    const observerOptions = {
        threshold: 0.1,
        rootMargin: '0px 0px -100px 0px'
    };

    const observer = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                entry.target.classList.add('visible');
            }
        });
    }, observerOptions);

    fadeElements.forEach(element => observer.observe(element));
}

/* ============================================
   SCROLL-TRIGGERED REVEAL ANIMATIONS
   ============================================ */

function initScrollReveal() {
    const revealElements = document.querySelectorAll('.reveal, .reveal-left, .reveal-right, .reveal-scale');
    if (revealElements.length === 0) return;

    // Skip animations if user prefers reduced motion
    if (prefersReducedMotion) {
        revealElements.forEach(el => el.classList.add('visible'));
        return;
    }

    const observerOptions = {
        threshold: 0.15,
        rootMargin: '0px 0px -50px 0px'
    };

    const observer = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                entry.target.classList.add('visible');
                // Optionally unobserve after animation (one-time reveal)
                observer.unobserve(entry.target);
            }
        });
    }, observerOptions);

    revealElements.forEach(element => observer.observe(element));
}

/* ============================================
   STICKY MOBILE CTA
   ============================================ */

function initStickyCTA() {
    const stickyCTA = document.querySelector('.sticky-cta-mobile');
    if (!stickyCTA) return;

    const showThreshold = 600; // Show after scrolling 600px
    let ticking = false;

    window.addEventListener('scroll', () => {
        if (!ticking) {
            window.requestAnimationFrame(() => {
                const currentScrollY = window.scrollY || document.documentElement.scrollTop;

                if (currentScrollY > showThreshold) {
                    stickyCTA.classList.add('visible');
                } else {
                    stickyCTA.classList.remove('visible');
                }
                ticking = false;
            });
            ticking = true;
        }
    }, { passive: true });
}

/* ============================================
   COUNTER ANIMATION (Alpine.js helper)
   ============================================ */

// This function can be called from Alpine.js x-init
window.animateCounter = function(element, target, duration = 2000) {
    if (prefersReducedMotion) {
        element.textContent = target.toLocaleString();
        return;
    }

    const start = 0;
    const startTime = performance.now();

    function update(currentTime) {
        const elapsed = currentTime - startTime;
        const progress = Math.min(elapsed / duration, 1);

        // Easing function (ease-out)
        const easeOut = 1 - Math.pow(1 - progress, 3);
        const current = Math.floor(start + (target - start) * easeOut);

        element.textContent = current.toLocaleString();

        if (progress < 1) {
            requestAnimationFrame(update);
        } else {
            element.textContent = target.toLocaleString();
        }
    }

    requestAnimationFrame(update);
};

/* ============================================
   SMOOTH SCROLL FOR ANCHOR LINKS
   ============================================ */

function initSmoothScroll() {
    document.querySelectorAll('a[href^="#"]').forEach(anchor => {
        anchor.addEventListener('click', function(e) {
            const targetId = this.getAttribute('href');
            if (targetId === '#') return;

            const target = document.querySelector(targetId);
            if (target) {
                e.preventDefault();
                target.scrollIntoView({
                    behavior: prefersReducedMotion ? 'auto' : 'smooth',
                    block: 'start'
                });
            }
        });
    });
}

/* ============================================
   ACCORDION FUNCTIONALITY
   ============================================ */

function initAccordions() {
    const accordionTriggers = document.querySelectorAll('.accordion-trigger');

    accordionTriggers.forEach(trigger => {
        trigger.addEventListener('click', () => {
            const content = trigger.nextElementSibling;
            const isOpen = content.classList.contains('open');

            // Close all other accordions (optional - remove for multi-open)
            document.querySelectorAll('.accordion-content').forEach(item => {
                if (item !== content) {
                    item.classList.remove('open');
                }
            });

            // Toggle current accordion
            if (isOpen) {
                content.classList.remove('open');
                trigger.setAttribute('aria-expanded', 'false');
            } else {
                content.classList.add('open');
                trigger.setAttribute('aria-expanded', 'true');
            }
        });
    });
}

/* ============================================
   INITIALIZE ALL SCRIPTS
   ============================================ */

document.addEventListener('DOMContentLoaded', () => {
    initHeroParallax();
    initFadeInObserver();
    initScrollReveal();
    initStickyCTA();
    initSmoothScroll();
    initAccordions();

    // Toast Notification Handler
    document.body.addEventListener('show-toast', function(evt) {
        const message = evt.detail.message || evt.detail.value || evt.detail;
        const type = evt.detail.type || 'success'; // success, error
        
        const toast = document.createElement('div');
        toast.className = `fixed top-4 right-4 z-[100] p-4 rounded-lg shadow-lg transform transition-all duration-300 translate-y-[-20px] opacity-0 ${
            type === 'error' ? 'bg-red-500 text-white' : 'bg-green-500 text-white'
        }`;
        toast.textContent = message;
        
        document.body.appendChild(toast);
        
        // Animate in
        requestAnimationFrame(() => {
            toast.classList.remove('translate-y-[-20px]', 'opacity-0');
        });
        
        // Remove after 3 seconds
        setTimeout(() => {
            toast.classList.add('translate-y-[-20px]', 'opacity-0');
            setTimeout(() => toast.remove(), 300);
        }, 3000);
    });

    // Modal Close Handler
    document.body.addEventListener('close-modal', function(evt) {
        const modalId = evt.detail.id || evt.detail.value || evt.detail;
        const modal = document.getElementById(modalId);
        if (modal) {
            modal.remove();
        }
    });

    // Auto-initialize Lucide icons after HTMX swaps
    document.body.addEventListener('htmx:afterSwap', function(evt) {
        if (typeof lucide !== 'undefined') {
            lucide.createIcons(evt.detail.target);
        }
    });
});

/* ============================================
   GLOBAL SECURITY & UI HELPERS (CSP Compliant)
   ============================================ */

// Template Editor Alpine Component
window.templateEditor = function() {
    return {
        content: document.getElementById("template-editor")?.innerHTML || "",
        updateContent(html) {
            this.content = html;
        },
        formatBold() {
            document.execCommand("bold", false, null);
        },
        formatItalic() {
            document.execCommand("italic", false, null);
        },
        formatHeading() {
            document.execCommand("formatBlock", false, "h2");
        },
        formatParagraph() {
            document.execCommand("formatBlock", false, "p");
        },
        formatList() {
            document.execCommand("insertUnorderedList", false, null);
        },
        insertVariable(variable) {
            const editor = document.getElementById("template-editor");
            if (editor) {
                editor.focus();
                document.execCommand("insertText", false, "{{ " + variable + " }}");
                this.content = editor.innerHTML;
            }
        }
    };
};

// Global Confirmation Modal Helper
let confirmationModalTriggerElement = null;

window.openConfirmationModal = function(options, triggerElement) {
    const modal = document.getElementById('confirmation-modal');
    const titleEl = document.getElementById('confirmation-modal-title');
    const msgEl = document.getElementById('confirmation-modal-message');
    const confirmBtn = document.getElementById('confirmation-modal-btn');

    if (!modal || !titleEl || !msgEl || !confirmBtn) {
        console.error('Confirmation modal elements not found');
        return;
    }

    // Store the element that triggered the modal for relative target resolution
    confirmationModalTriggerElement = triggerElement || (window.event ? window.event.currentTarget : null);

    if (options.title) titleEl.textContent = options.title;
    if (options.message) msgEl.textContent = options.message;

    // Replace button to remove old event listeners
    const newBtn = confirmBtn.cloneNode(true);
    confirmBtn.parentNode.replaceChild(newBtn, confirmBtn);

    newBtn.addEventListener('click', function() {
        if (options.confirmUrl) {
            // Resolve target relative to the original trigger element
            let targetEl = null;
            if (options.target && confirmationModalTriggerElement) {
                if (options.target.startsWith('closest ')) {
                    const selector = options.target.replace('closest ', '');
                    targetEl = confirmationModalTriggerElement.closest(selector);
                } else {
                    targetEl = document.querySelector(options.target);
                }
            }

            htmx.ajax(options.confirmMethod || 'DELETE', options.confirmUrl, {
                target: targetEl || options.target,
                swap: options.swap || 'innerHTML'
            });
        }
        window.closeConfirmationModal();
    });

    modal.classList.remove('hidden');
};

window.openConfirmationModalFromData = function(el) {
    window.openConfirmationModal({
        title: el.getAttribute('data-confirm-title'),
        message: el.getAttribute('data-confirm-message'),
        confirmUrl: el.getAttribute('data-confirm-url'),
        confirmMethod: el.getAttribute('data-confirm-method') || 'DELETE',
        target: el.getAttribute('data-confirm-target'),
        swap: el.getAttribute('data-confirm-swap')
    }, el);
};

window.closeConfirmationModal = function() {
    const modal = document.getElementById('confirmation-modal');
    if (modal) modal.classList.add('hidden');
    confirmationModalTriggerElement = null;
};

// Calendar Helpers
window.initCalendar = function() {
    try {
        var calendarEl = document.getElementById('calendar');
        if (!calendarEl) return;

        if (typeof FullCalendar === 'undefined') {
            console.error('FullCalendar is not defined! Check script loading.');
            return;
        }

        const locale = calendarEl.getAttribute('data-locale') || 'en';

        function getInitialView() {
            if (window.innerWidth < 640) return 'listWeek';
            if (window.innerWidth < 768) return 'timeGridDay';
            return 'timeGridWeek';
        }

        function getToolbarConfig() {
            if (window.innerWidth < 640) {
                return { left: 'prev,next', center: 'title', right: 'today' };
            }
            if (window.innerWidth < 768) {
                return { left: 'prev,next today', center: 'title', right: 'timeGridDay,listWeek' };
            }
            return { left: 'prev,next today', center: 'title', right: 'dayGridMonth,timeGridWeek,timeGridDay,listWeek' };
        }

        var calendar = new FullCalendar.Calendar(calendarEl, {
            initialView: getInitialView(),
            locale: locale,
            headerToolbar: getToolbarConfig(),
            themeSystem: 'standard',
            height: '100%',
            slotMinTime: '08:00:00',
            slotMaxTime: '20:00:00',
            allDaySlot: false,
            navLinks: true,
            nowIndicator: true,
            events: '/api/calendar/events',
            windowResize: function() {
                try {
                    const newView = getInitialView();
                    if (calendar.view.type !== newView) {
                        calendar.changeView(newView);
                    }
                    calendar.setOption('headerToolbar', getToolbarConfig());
                } catch (e) {
                    console.error('Error in windowResize:', e);
                }
            },
            eventClick: function(info) {
                window.openEventModal(info.event);
            }
        });
        
        calendarEl._fullCalendar = calendar;
        calendar.render();
    } catch (e) {
        console.error('CRITICAL ERROR initializing calendar:', e);
    }
};

window.openEventModal = function(event) {
    const modal = document.getElementById('event-detail-modal');
    const props = event.extendedProps;
    if (!modal) return;
    
    modal.dataset.eventId = event.id;
    document.getElementById('modal-event-title').textContent = event.title;
    document.getElementById('modal-event-client').textContent = props.clientName || '-';
    document.getElementById('modal-event-lawyer').textContent = props.lawyer || '-';
    
    const statusEl = document.getElementById('modal-event-status');
    statusEl.textContent = props.status || '-';
    statusEl.className = 'badge badge-lg font-bold rounded-sm';
    if (props.status === 'Scheduled' || props.status === 'Confirmed') {
        statusEl.classList.add('badge-success', 'text-white');
    } else if (props.status === 'Cancelled') {
        statusEl.classList.add('badge-error', 'text-white');
    } else {
        statusEl.classList.add('badge-neutral', 'text-white');
    }

    const startTime = event.start ? event.start.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) : '';
    const endTime = event.end ? event.end.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) : '';
    const dateStr = event.start ? event.start.toLocaleDateString([], { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' }) : '';
    document.getElementById('modal-event-time').textContent = `${dateStr}, ${startTime} - ${endTime}`;

    const notesContainer = document.getElementById('modal-notes-container');
    if (props.notes && props.notes.trim()) {
        document.getElementById('modal-event-notes').textContent = props.notes;
        notesContainer.classList.remove('hidden');
    } else {
        notesContainer.classList.add('hidden');
    }

    modal.classList.remove('hidden');
    modal.classList.add('flex');
    document.body.style.overflow = 'hidden';
};

window.closeEventModal = function() {
    const modal = document.getElementById('event-detail-modal');
    if (modal) {
        modal.classList.add('hidden');
        modal.classList.remove('flex');
        document.body.style.overflow = '';
    }
};

window.cancelEvent = function() {
    const modal = document.getElementById('event-detail-modal');
    const id = modal.dataset.eventId;
    if(!id) return;

    fetch('/api/appointments/' + id + '/cancel', {
        method: 'POST',
        headers: {
            'X-CSRF-Token': document.querySelector('meta[name="csrf-token"]')?.getAttribute('content')
        }
    })
    .then(response => {
        if(!response.ok) throw new Error('Failed to cancel');
        return response.json();
    })
    .then(() => {
        window.closeEventModal();
        var calendarEl = document.getElementById('calendar');
        var calendar = calendarEl?._fullCalendar; 
        if (calendar) {
            calendar.refetchEvents();
        } else {
            window.location.reload();
        }
    })
    .catch(err => {
        console.error(err);
        alert('Error cancelling appointment');
    });
};

// Appointment Page Alpine Helper
window.appointmentPage = function() {
    return {
        showCreateModal: false,
        selectedCase: '',
        selectedLawyer: '',
        selectedDate: '',
        selectedStartTime: '',
        selectedEndTime: '',
        slots: [],
        casesLoaded: false,
        lawyersLoaded: false,
        
        init() {
            this.$watch('showCreateModal', (value) => {
                if (value) {
                    this.loadCases();
                    this.loadLawyers();
                }
            });
        },
        
        loadCases() {
            if (this.casesLoaded) return;
            const select = document.getElementById('case-select');
            if (!select) return;
            
            fetch('/api/appointments/cases')
                .then(res => res.text())
                .then(html => {
                    select.innerHTML = html;
                    this.casesLoaded = true;
                })
                .catch(err => console.error('Error loading cases:', err));
        },
        
        loadLawyers() {
            if (this.lawyersLoaded) return;
            const select = document.getElementById('lawyer-select');
            if (!select) return;
            
            fetch('/api/appointments/lawyers')
                .then(res => res.text())
                .then(html => {
                    select.innerHTML = html;
                    this.lawyersLoaded = true;
                })
                .catch(err => console.error('Error loading lawyers:', err));
        },
        
        loadSlots() {
            if (!this.selectedDate) return;
            
            const lawyerId = this.selectedLawyer || document.querySelector('input[name="lawyer_id"]')?.value;
            if (!lawyerId) return;
            
            fetch(`/api/appointments/slots?date=${this.selectedDate}&lawyer_id=${lawyerId}`)
                .then(res => res.json())
                .then(data => {
                    this.slots = data.slots || [];
                    this.renderSlots();
                });
        },
        
        renderSlots() {
            const container = document.getElementById('available-slots');
            if (!container) return;
            
            if (this.slots.length === 0) {
                container.innerHTML = '<p class="col-span-3 text-base-content/50 text-sm italic py-2">No available slots for this date</p>';
                return;
            }
            
            container.innerHTML = this.slots.map(slot => {
                const start = new Date(slot.start_time);
                const timeStr = start.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
                const isSelected = this.selectedStartTime === slot.start_time;
                return `
                    <button 
                        type="button"
                        @click="selectSlot('${slot.start_time}', '${slot.end_time}')"
                        class="btn btn-sm rounded-sm ${isSelected ? 'btn-primary' : 'btn-outline border-base-300'}"
                    >${timeStr}</button>
                `;
            }).join('');
            
            // Re-init Alpine on new elements
            if (window.Alpine) {
                window.Alpine.initTree(container);
            }
        },
        
        selectSlot(start, end) {
            this.selectedStartTime = start;
            this.selectedEndTime = end;
            this.renderSlots();
        }
    };
};




