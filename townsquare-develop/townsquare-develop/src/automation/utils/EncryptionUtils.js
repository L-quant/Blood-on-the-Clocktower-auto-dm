/**
 * 加密工具
 * 提供简单的加密/解密功能用于保护敏感信息
 */

/**
 * 简单的XOR加密/解密
 * 注意：这是一个简单的实现，不适用于高安全性需求
 * @param {string} text - 要加密/解密的文本
 * @param {string} key - 密钥
 * @returns {string} 加密/解密后的文本
 */
function xorEncrypt(text, key) {
  if (!text || !key) {
    return text;
  }

  let result = '';
  for (let i = 0; i < text.length; i++) {
    const charCode = text.charCodeAt(i) ^ key.charCodeAt(i % key.length);
    result += String.fromCharCode(charCode);
  }
  return result;
}

/**
 * Base64编码
 * @param {string} text - 要编码的文本
 * @returns {string} 编码后的文本
 */
function base64Encode(text) {
  if (typeof btoa !== 'undefined') {
    return btoa(unescape(encodeURIComponent(text)));
  }
  // Node.js环境
  return Buffer.from(text, 'utf-8').toString('base64');
}

/**
 * Base64解码
 * @param {string} encoded - 编码的文本
 * @returns {string} 解码后的文本
 */
function base64Decode(encoded) {
  if (typeof atob !== 'undefined') {
    return decodeURIComponent(escape(atob(encoded)));
  }
  // Node.js环境
  return Buffer.from(encoded, 'base64').toString('utf-8');
}

/**
 * 加密工具类
 */
export default class EncryptionUtils {
  /**
   * 生成随机密钥
   * @param {number} length - 密钥长度
   * @returns {string} 随机密钥
   */
  static generateKey(length = 16) {
    const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
    let key = '';
    for (let i = 0; i < length; i++) {
      key += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    return key;
  }

  /**
   * 加密文本
   * @param {string} text - 要加密的文本
   * @param {string} key - 密钥
   * @returns {string} 加密后的文本（Base64编码）
   */
  static encrypt(text, key) {
    if (!text) {
      return text;
    }

    try {
      // 使用XOR加密
      const encrypted = xorEncrypt(text, key);
      // Base64编码
      return base64Encode(encrypted);
    } catch (error) {
      console.error('[EncryptionUtils] Encryption failed:', error);
      return text;
    }
  }

  /**
   * 解密文本
   * @param {string} encryptedText - 加密的文本（Base64编码）
   * @param {string} key - 密钥
   * @returns {string} 解密后的文本
   */
  static decrypt(encryptedText, key) {
    if (!encryptedText) {
      return encryptedText;
    }

    try {
      // Base64解码
      const decoded = base64Decode(encryptedText);
      // XOR解密
      return xorEncrypt(decoded, key);
    } catch (error) {
      console.error('[EncryptionUtils] Decryption failed:', error);
      return encryptedText;
    }
  }

  /**
   * 加密对象
   * @param {object} obj - 要加密的对象
   * @param {string} key - 密钥
   * @returns {string} 加密后的JSON字符串
   */
  static encryptObject(obj, key) {
    try {
      const json = JSON.stringify(obj);
      return this.encrypt(json, key);
    } catch (error) {
      console.error('[EncryptionUtils] Object encryption failed:', error);
      return null;
    }
  }

  /**
   * 解密对象
   * @param {string} encryptedJson - 加密的JSON字符串
   * @param {string} key - 密钥
   * @returns {object} 解密后的对象
   */
  static decryptObject(encryptedJson, key) {
    try {
      const json = this.decrypt(encryptedJson, key);
      return JSON.parse(json);
    } catch (error) {
      console.error('[EncryptionUtils] Object decryption failed:', error);
      return null;
    }
  }

  /**
   * 加密游戏状态中的敏感信息
   * @param {object} gameState - 游戏状态
   * @param {string} key - 密钥
   * @returns {object} 加密后的游戏状态
   */
  static encryptGameState(gameState, key) {
    if (!gameState || !key) {
      return gameState;
    }

    const encrypted = { ...gameState };

    // 加密玩家角色信息
    if (encrypted.players) {
      encrypted.players = encrypted.players.map(player => {
        const encryptedPlayer = { ...player };
        
        if (player.role) {
          // 将角色信息转换为JSON并加密
          encryptedPlayer.role = this.encrypt(JSON.stringify(player.role), key);
          encryptedPlayer._roleEncrypted = true;
        }

        return encryptedPlayer;
      });
    }

    encrypted._encrypted = true;
    return encrypted;
  }

  /**
   * 解密游戏状态中的敏感信息
   * @param {object} encryptedGameState - 加密的游戏状态
   * @param {string} key - 密钥
   * @returns {object} 解密后的游戏状态
   */
  static decryptGameState(encryptedGameState, key) {
    if (!encryptedGameState || !encryptedGameState._encrypted || !key) {
      return encryptedGameState;
    }

    const decrypted = { ...encryptedGameState };

    // 解密玩家角色信息
    if (decrypted.players) {
      decrypted.players = decrypted.players.map(player => {
        const decryptedPlayer = { ...player };
        
        if (player._roleEncrypted && player.role) {
          try {
            const roleJson = this.decrypt(player.role, key);
            decryptedPlayer.role = JSON.parse(roleJson);
            delete decryptedPlayer._roleEncrypted;
          } catch (error) {
            console.error('[EncryptionUtils] Failed to decrypt player role:', error);
          }
        }

        return decryptedPlayer;
      });
    }

    delete decrypted._encrypted;
    return decrypted;
  }

  /**
   * 生成哈希值（简单实现）
   * @param {string} text - 要哈希的文本
   * @returns {string} 哈希值
   */
  static hash(text) {
    if (!text) {
      return '';
    }

    let hash = 0;
    for (let i = 0; i < text.length; i++) {
      const char = text.charCodeAt(i);
      hash = ((hash << 5) - hash) + char;
      hash = hash & hash; // Convert to 32bit integer
    }
    return hash.toString(36);
  }

  /**
   * 验证哈希值
   * @param {string} text - 原始文本
   * @param {string} hash - 哈希值
   * @returns {boolean} 是否匹配
   */
  static verifyHash(text, hash) {
    return this.hash(text) === hash;
  }
}
