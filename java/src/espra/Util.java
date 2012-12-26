// Public Domain (-) 2012 The Espra Authors.
// See the Espra UNLICENSE file for details.

package espra;

public class Util {
	public static boolean isAuthKey(String stringKey) {
		if (stringKey == null) {
			return false;
		}
		byte[] key = stringKey.getBytes();
		if (key.length != Config.Key.length) {
			return false;
		}
		byte total = 0;
		for (int i = 0; i < key.length; i++) {
			total |= key[i] ^ Config.Key[i];
		}
		return (total == 0);
	}
}
