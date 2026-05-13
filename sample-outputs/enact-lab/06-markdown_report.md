# Funding Pipeline - ENACT Lab

## Summary

The refreshed ledger found one medium-confidence NIH opportunity and one watch
lead for ENACT Lab. It also avoided recommending SBIR/STTR records, which the
brief explicitly excludes.

- Current recommendation count: 1 medium lead, 1 watch lead
- Retrieval backend: fts5
- CLI `no_llm`: true
- Strongest issue: the best matches still need NOFO review for psychedelic-specific language and Yale/PI eligibility.

## Recommended Opportunities

### 1. [Addressing Methodological Challenges with Clinical Trials of Rapid-Acting Psychotropic Interventional Drugs (RAPIDs)](https://www.grants.gov/search-results-detail/359202)

- Agency/source: National Institutes of Health
- Confidence: medium
- Deadline: 10/11/2026
- Why this fits: rapid-acting psychotropic interventional drugs and clinical-trial methodology are close to ENACT's psychedelic psychiatry focus.
- Caveat: packet evidence does not mention psychedelics specifically.
- Next step: map ENACT's psilocybin, MDMA, or ketamine trial methods to the NOFO's methodological-challenge language before contacting the program officer.

<details>
<summary>Evidence</summary>

- Source: grants-gov-api
- URL: https://www.grants.gov/search-results-detail/359202
- Claim: NIH posted the RAPIDs R01 opportunity with clinical trial required.

</details>

### 2. [Exploratory Clinical Neuroscience Research on Substance Use Disorders](https://www.grants.gov/search-results-detail/361518)

- Agency/source: National Institutes of Health
- Confidence: watch
- Deadline: none in current evidence
- Why this fits: clinical neuroscience is a direct domain match, and substance-use research can be adjacent to psychedelic medicine mechanisms.
- Caveat: packet evidence is forecast-level and does not show psychedelic, PTSD, depression, or OCD language.
- Next step: monitor the final NOFO and verify whether it includes mechanisms, imaging, or treatment-resistant psychiatric indications relevant to ENACT.

## Not Recommended

| Candidate | Why rejected |
|---|---|
| NIH Blueprint and BRAIN Initiative ACTION Potential Program | Training/career-stage program, not direct project funding for the lab's current studies. |
| Analgesics, Anesthetics, and Addiction Clinical Trials | Biomedical, but packet evidence points away from psychedelic psychiatry. |
| NIMH Research Education Programs for Psychiatry Residents | Education mechanism, not a research project opportunity for the lab. |

## Suggested Next Search

- NIMH, NIDA, and NCCIH psychedelic psychiatry and mechanism-of-action opportunities.
- PCORI and ARPA-H clinical-research programs.
- DoD CDMRP PTSD/PRMRP opportunities.
- Yale and Connecticut research infrastructure or trainee support.

## Coverage

| Source lane | Status | Note |
|---|---|---|
| Grants.gov | matched | canonical federal opportunity lane |
| SBIR/STTR | matched | small business funding lane, but not appropriate for this assignment |
| ARPA-E | checked_no_match | No current ARPA-E programs match |
| DOE EERE | checked_no_match | energy funding lane |
| NSF | matched | research and commercialization lane |
| state economic development: Connecticut | checked_no_match | state-specific source lane required by assignment geography |

## Negative Evidence

- SBIR/STTR: present in the ledger but rejected by assignment constraints.
- ARPA-E: No current ARPA-E programs match.
- Connecticut state economic development: no matching current evidence in the ledger.

## Provenance

Generated at 2026-05-13 from a deterministic `grant-finder research` packet. The CLI reported `retrieval.no_llm: true`; final recommendation judgment was made by the OpenProse `rank-opportunities` service.
