import DOMPurify from 'dompurify';
import { JSDOM } from 'jsdom';

const window = new JSDOM('').window;
const purify = DOMPurify(window);

export class InputSanitizer {
  static sanitizeHtml(input: string): string {
    // Configure DOMPurify with strict settings
    const config = {
      ALLOWED_TAGS: ['b', 'i', 'em', 'strong', 'p', 'br'],
      ALLOWED_ATTR: [],
      KEEP_CONTENT: true,
      RETURN_DOM: false,
      RETURN_DOM_FRAGMENT: false,
      RETURN_DOM_IMPORT: false
    };
    
    return purify.sanitize(input, config);
  }

  static sanitizeUserInput(input: string): string {
    // Remove potentially dangerous characters
    let sanitized = input.replace(/[<>"'&]/g, '');
    
    // Trim whitespace
    sanitized = sanitized.trim();
    
    // Limit length
    if (sanitized.length > 1000) {
      sanitized = sanitized.substring(0, 1000);
    }
    
    return sanitized;
  }

  static validateAndSanitizeEmail(email: string): string {
    // Basic email validation
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!emailRegex.test(email)) {
      throw new Error('Invalid email format');
    }
    
    // Sanitize email
    return email.toLowerCase().trim();
  }

  static sanitizeFilename(filename: string): string {
    // Remove dangerous characters from filename
    return filename.replace(/[^a-zA-Z0-9._-]/g, '_');
  }
}