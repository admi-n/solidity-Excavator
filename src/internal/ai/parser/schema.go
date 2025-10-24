package parser

// GetExpectedJSONSchema 返回期望的 JSON 响应格式说明
func GetExpectedJSONSchema() string {
	return `{
  "contract_address": "0x...",
  "vulnerabilities": [
    {
      "type": "Reentrancy|Integer Overflow|Access Control|...",
      "severity": "Critical|High|Medium|Low",
      "description": "Detailed description of the vulnerability",
      "location": "Function or contract name",
      "line_numbers": [10, 15, 20],
      "code_snippet": "Relevant code snippet",
      "impact": "Potential impact of this vulnerability",
      "remediation": "How to fix this vulnerability",
      "references": ["https://swcregistry.io/docs/SWC-107"],
      "swc_id": "SWC-107"
    }
  ],
  "summary": "Overall security assessment summary",
  "risk_score": 7.5,
  "recommendations": [
    "Recommendation 1",
    "Recommendation 2"
  ]
}`
}

// GetSchemaInstructions 返回给 AI 的格式说明
func GetSchemaInstructions() string {
	return `Please analyze the smart contract and return your findings in the following JSON format:

` + GetExpectedJSONSchema() + `

Requirements:
1. Identify ALL potential vulnerabilities in the contract
2. For each vulnerability:
   - Specify the type (e.g., Reentrancy, Integer Overflow, Access Control)
   - Assign severity: Critical (can lead to fund loss), High (serious security issue), Medium (potential issue), Low (minor issue)
   - Provide detailed description
   - Include the location (function name, line numbers if possible)
   - Include relevant code snippet
   - Explain the potential impact
   - Provide remediation steps
   - Reference SWC Registry IDs where applicable
3. Provide an overall summary of the contract's security posture
4. Calculate a risk score from 0-10 (10 being most risky)
5. List prioritized recommendations for improvement

Return ONLY the JSON object, without any additional text or markdown formatting.`
}

// SeverityLevel 定义严重性级别
type SeverityLevel string

const (
	SeverityCritical SeverityLevel = "Critical"
	SeverityHigh     SeverityLevel = "High"
	SeverityMedium   SeverityLevel = "Medium"
	SeverityLow      SeverityLevel = "Low"
	SeverityInfo     SeverityLevel = "Info"
)

// GetSeverityScore 获取严重性分数（用于排序）
func GetSeverityScore(severity string) int {
	switch SeverityLevel(severity) {
	case SeverityCritical:
		return 5
	case SeverityHigh:
		return 4
	case SeverityMedium:
		return 3
	case SeverityLow:
		return 2
	case SeverityInfo:
		return 1
	default:
		return 0
	}
}

// CommonVulnerabilityTypes 常见漏洞类型列表
var CommonVulnerabilityTypes = []string{
	"Reentrancy",
	"Integer Overflow",
	"Integer Underflow",
	"Unchecked External Call",
	"Access Control",
	"Denial of Service",
	"Timestamp Dependence",
	"Front Running",
	"Delegatecall to Untrusted Callee",
	"Unprotected Selfdestruct",
	"Uninitialized Storage Pointer",
	"Floating Pragma",
	"Outdated Compiler Version",
	"Use of Deprecated Functions",
	"Unsafe Type Inference",
	"Block Gas Limit",
	"Transaction Order Dependence",
	"Authorization through tx.origin",
	"Signature Malleability",
	"Insufficient Gas Griefing",
	"State Variable Default Visibility",
	"Off-By-One",
	"Lack of Proper Signature Verification",
	"Requirement Violation",
	"Write to Arbitrary Storage Location",
	"Incorrect Constructor Name",
	"Shadowing State Variables",
	"Weak Sources of Randomness",
	"Missing Protection against Signature Replay Attacks",
}
