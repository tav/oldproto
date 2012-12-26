// Public Domain (-) 2011-2012 The Espra Authors.
// See the Espra UNLICENSE file for details.

package espra;

import java.io.IOException;
import java.net.URL;
import javax.servlet.http.*;

import de.l3s.boilerpipe.BoilerpipeExtractor;
import de.l3s.boilerpipe.BoilerpipeProcessingException;
import de.l3s.boilerpipe.extractors.CommonExtractors;
import de.l3s.boilerpipe.sax.HTMLHighlighter;

public class ExtractorServlet extends HttpServlet {
	public void doGet(HttpServletRequest req, HttpServletResponse resp)
			throws IOException {
		final BoilerpipeExtractor extractor = CommonExtractors.ARTICLE_EXTRACTOR;
		final HTMLHighlighter hh = HTMLHighlighter.newExtractingInstance();
		resp.setContentType("text/html; charset=utf-8");
		String response;
		try {
			if (Util.isAuthKey(req.getParameter("key"))) {
				URL url = new URL(req.getParameter("url"));
				response = hh.process(url, extractor);
			} else {
				response = "ERROR: Invalid auth key.";
			};
		} catch (Exception e) {
			response = "ERROR: " + e.toString();
		}
		resp.getWriter().println(response);
	}
}
