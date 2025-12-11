// Mobile menu toggle
const mobileMenuBtn = document.querySelector('.mobile-menu-btn');
const navLinks = document.querySelector('.nav-links');

if (mobileMenuBtn) {
    mobileMenuBtn.addEventListener('click', () => {
        navLinks.classList.toggle('active');
        mobileMenuBtn.classList.toggle('active');
    });
}

// Copy button functionality
document.querySelectorAll('.copy-btn').forEach(btn => {
    btn.addEventListener('click', async () => {
        const code = btn.closest('.code-block').querySelector('code').textContent;

        try {
            await navigator.clipboard.writeText(code);

            // Visual feedback
            const svg = btn.querySelector('svg');
            const originalHTML = svg.outerHTML;

            svg.outerHTML = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="20 6 9 17 4 12"></polyline>
            </svg>`;

            btn.style.borderColor = 'var(--accent-green)';
            btn.querySelector('svg').style.stroke = 'var(--accent-green)';

            setTimeout(() => {
                btn.querySelector('svg').outerHTML = originalHTML;
                btn.style.borderColor = '';
            }, 2000);
        } catch (err) {
            console.error('Failed to copy:', err);
        }
    });
});

// Smooth scroll for anchor links
document.querySelectorAll('a[href^="#"]').forEach(anchor => {
    anchor.addEventListener('click', function (e) {
        e.preventDefault();
        const target = document.querySelector(this.getAttribute('href'));
        if (target) {
            target.scrollIntoView({
                behavior: 'smooth',
                block: 'start'
            });
        }
    });
});

// Navbar background on scroll
const navbar = document.querySelector('.navbar');
let lastScroll = 0;

window.addEventListener('scroll', () => {
    const currentScroll = window.pageYOffset;

    if (currentScroll > 100) {
        navbar.style.background = 'rgba(10, 10, 15, 0.95)';
    } else {
        navbar.style.background = 'rgba(10, 10, 15, 0.8)';
    }

    lastScroll = currentScroll;
});

// Intersection Observer for animations
const observerOptions = {
    threshold: 0.1,
    rootMargin: '0px 0px -50px 0px'
};

const observer = new IntersectionObserver((entries) => {
    entries.forEach(entry => {
        if (entry.isIntersecting) {
            entry.target.classList.add('animate-in');
            observer.unobserve(entry.target);
        }
    });
}, observerOptions);

// Observe elements for animation
document.querySelectorAll('.feature-card, .service-category, .step').forEach(el => {
    el.style.opacity = '0';
    el.style.transform = 'translateY(20px)';
    el.style.transition = 'opacity 0.5s ease, transform 0.5s ease';
    observer.observe(el);
});

// Add animate-in class styles
const style = document.createElement('style');
style.textContent = `
    .animate-in {
        opacity: 1 !important;
        transform: translateY(0) !important;
    }
`;
document.head.appendChild(style);

// Typing effect for code blocks (optional enhancement)
function typeCode(element, code, speed = 30) {
    let i = 0;
    element.innerHTML = '';

    function type() {
        if (i < code.length) {
            element.innerHTML = code.substring(0, i + 1);
            i++;
            setTimeout(type, speed);
        }
    }

    type();
}

// Console greeting
console.log('%c Infra Operator ', 'background: linear-gradient(135deg, #FF9900, #FF6600); color: white; font-size: 20px; padding: 10px 20px; border-radius: 8px; font-weight: bold;');
console.log('%c Kubernetes Operator for AWS Infrastructure ', 'color: #a0a0b0; font-size: 14px;');
console.log('%c https://github.com/andrebassi/infra-operator ', 'color: #FF9900; font-size: 12px;');
