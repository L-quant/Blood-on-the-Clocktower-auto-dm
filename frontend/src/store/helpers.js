/**
 * Clean a role ID to lowercase alphanumeric
 */
export const cleanId = id => id.toLocaleLowerCase().replace(/[^a-z0-9]/g, "");

/**
 * Detect if the current viewport is mobile
 */
export const isMobile = () => window.innerWidth <= 768;

/**
 * Generate a random ID
 */
export const randomId = () => Math.random().toString(36).substr(2, 10);

/**
 * Format seat index for display (1-based)
 */
export const seatLabel = (seatIndex) => `${seatIndex}号`;
