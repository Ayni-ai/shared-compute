import { ImageResponse } from "next/og";

// The link-preview image for founding-member pages (1200x630), drawn at build time so
// the repository carries no binary. The API's /m/{code} share pages and the site's
// /founding page point here.
export const dynamic = "force-static";

const navy = "#0d1638";
const teal = "#0f6b63";
const mint = "#7fd3c9";

export function GET() {
  return new ImageResponse(
    (
      <div style={{ width: "100%", height: "100%", display: "flex", background: `linear-gradient(135deg, #14204d, ${navy})`, color: "#fff", padding: "80px 80px" }}>
        <div style={{ display: "flex", flexDirection: "column", flex: 1 }}>
          <div style={{ fontSize: 34, letterSpacing: 10 }}>AYNI</div>
          <div style={{ fontSize: 26, letterSpacing: 5, color: mint, marginTop: 92 }}>FOUNDING MEMBERS</div>
          <div style={{ fontSize: 120, lineHeight: 1, marginTop: 8 }}>1,000</div>
          <div style={{ fontSize: 30, marginTop: 18 }}>places. Numbered. Verified by the network.</div>
          <div style={{ fontSize: 24, color: "#cfd6ea", marginTop: 54 }}>A month of test jobs on a device you are not using,</div>
          <div style={{ fontSize: 24, color: "#cfd6ea" }}>and one honest piece of feedback. ayni-ai.com/founding</div>
        </div>
        <div style={{ display: "flex", alignItems: "center", justifyContent: "center", width: 380 }}>
          <div style={{ display: "flex", alignItems: "center", justifyContent: "center", width: 380, height: 380, borderRadius: 190, border: "2px solid rgba(31,158,148,.35)" }}>
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center", width: 250, height: 250, borderRadius: 125, background: teal, border: `3px solid ${mint}` }}>
              <div style={{ fontSize: 18, letterSpacing: 4, color: mint }}>FOUNDING</div>
              <div style={{ fontSize: 34, marginTop: 6 }}>#1 to #1,000</div>
              <div style={{ fontSize: 18, letterSpacing: 4, color: mint, marginTop: 6 }}>TEST MONTH</div>
            </div>
          </div>
        </div>
      </div>
    ),
    { width: 1200, height: 630 },
  );
}
