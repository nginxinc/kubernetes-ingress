package k8s

import (
	"fmt"
	"sort"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const timeLayout = time.RFC3339

// reasons for invalidity
const failedValidationErrorMsg = "Validation Failed"
const missingUserSigErrorMsg = "Policy has unsatisfied signature requirements"
const duplicatedTagsErrorMsg = "Duplicate tag set"
const invalidTimestampErrorMsg = "Invalid timestamp"

// AppProtectUserSigChange holds resources that are affected by changes in UserSigs
type AppProtectUserSigChange struct {
	PolicyDeletions     []*unstructured.Unstructured
	PolicyAddsOrUpdates []*unstructured.Unstructured
	UserSigs            []*unstructured.Unstructured
}

// AppProtectChange represents a change in an App Protect resource
type AppProtectChange struct {
	// Op is an operation that needs be performed on the resource.
	Op Operation
	// Resource is the target resource.
	Resource interface{}
}

// AppProtectProblem represents a problem with an App Protect resource
type AppProtectProblem struct {
	// Object is a configuration object.
	Object *unstructured.Unstructured
	// Reason tells the reason. It matches the reason in the events of our configuration objects.
	Reason string
	// Messages gives the details about the problem. It matches the message in the events of our configuration objects.
	Message string
}

// AppProtectConfiguration holds representations of App Protect cluster resources
type AppProtectConfiguration struct {
	Policies map[string]*AppProtectPolicyEx
	LogConfs map[string]*AppProtectLogConfEx
	UserSigs map[string]*AppProtectUserSigEx
}

// NewAppProtectConfiguration creates a new AppProtectConfiguration
func NewAppProtectConfiguration() *AppProtectConfiguration {
	return &AppProtectConfiguration{
		Policies: make(map[string]*AppProtectPolicyEx),
		LogConfs: make(map[string]*AppProtectLogConfEx),
		UserSigs: make(map[string]*AppProtectUserSigEx),
	}
}

// AppProtectPolicyEx represents an App Protect policy cluster resource
type AppProtectPolicyEx struct {
	Obj           *unstructured.Unstructured
	SignatureReqs []SignatureReq
	IsValid       bool
	ErrorMsg      string
}

func (pol *AppProtectPolicyEx) setInvalid(reason string) {
	pol.IsValid = false
	pol.ErrorMsg = reason
}

func (pol *AppProtectPolicyEx) setValid() {
	pol.IsValid = true
	pol.ErrorMsg = ""
}

// SignatureReq describes a signature that is Requiered by the policy
type SignatureReq struct {
	Tag      string
	RevTimes *RevTimes
}

// RevTimes are requirements for signature revision time
type RevTimes struct {
	MinRevTime *time.Time
	MaxRevTime *time.Time
}

// AppProtectLogConfEx represents an App Protect Log Configuration cluster resource
type AppProtectLogConfEx struct {
	Obj      *unstructured.Unstructured
	IsValid  bool
	ErrorMsg string
}

// AppProtectUserSigEx represents an App Protect User Defined Signature cluster resource
type AppProtectUserSigEx struct {
	Obj      *unstructured.Unstructured
	Tag      string
	RevTime  *time.Time
	IsValid  bool
	ErrorMsg string
}

func (sig *AppProtectUserSigEx) setInvalid(reason string) {
	sig.IsValid = false
	sig.ErrorMsg = reason
}

func (sig *AppProtectUserSigEx) setValid() {
	sig.IsValid = true
	sig.ErrorMsg = ""
}

type appProtectUserSigSlice []*AppProtectUserSigEx

func (s appProtectUserSigSlice) Len() int {
	return len(s)
}

func (s appProtectUserSigSlice) Less(i, j int) bool {
	if s[i].Obj.GetCreationTimestamp().Time.Equal(s[j].Obj.GetCreationTimestamp().Time) {
		return s[i].Obj.GetUID() > s[j].Obj.GetUID()
	}
	return s[i].Obj.GetCreationTimestamp().Time.Before(s[j].Obj.GetCreationTimestamp().Time)
}

func (s appProtectUserSigSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func createAppProtectPolicyEx(policyObj *unstructured.Unstructured) (*AppProtectPolicyEx, error) {
	err := ValidateAppProtectPolicy(policyObj)
	if err != nil {
		errMsg := fmt.Sprintf("Error validating policy %s: %v", policyObj.GetName(), err)
		return &AppProtectPolicyEx{Obj: policyObj, IsValid: false, ErrorMsg: failedValidationErrorMsg}, fmt.Errorf(errMsg)
	}
	sigReqs := []SignatureReq{}
	// Check if policy has signature requirement (revision timestamp) and map them to tags
	list, found, err := unstructured.NestedSlice(policyObj.Object, "spec", "policy", "signature-requirements")
	if err != nil {
		errMsg := fmt.Sprintf("Error retrieving Signature requirements from %s: %v", policyObj.GetName(), err)
		return &AppProtectPolicyEx{Obj: policyObj, IsValid: false, ErrorMsg: failedValidationErrorMsg}, fmt.Errorf(errMsg)
	}
	if found {
		for _, req := range list {
			requirement := req.(map[string]interface{})
			if reqTag, ok := requirement["tag"]; ok {
				timeReq, err := buildRevTimes(requirement)
				if err != nil {
					errMsg := fmt.Sprintf("Error creating time requirements from %s: %v", policyObj.GetName(), err)
					return &AppProtectPolicyEx{Obj: policyObj, IsValid: false, ErrorMsg: invalidTimestampErrorMsg}, fmt.Errorf(errMsg)
				}
				sigReqs = append(sigReqs, SignatureReq{Tag: reqTag.(string), RevTimes: &timeReq})
			}
		}
	}
	return &AppProtectPolicyEx{
		Obj:           policyObj,
		SignatureReqs: sigReqs,
		IsValid:       true,
	}, nil
}

func buildRevTimes(requirement map[string]interface{}) (RevTimes, error) {
	timeReq := RevTimes{}
	if minRev, ok := requirement["minRevisionDatetime"]; ok {
		minRevTime, err := time.Parse(timeLayout, minRev.(string))
		if err != nil {
			errMsg := fmt.Sprintf("Error Parsing time from minRevisionDatetime %v", err)
			return timeReq, fmt.Errorf(errMsg)
		}
		timeReq.MinRevTime = &minRevTime
	}
	if maxRev, ok := requirement["maxRevisionDatetime"]; ok {
		maxRevTime, err := time.Parse(timeLayout, maxRev.(string))
		if err != nil {
			errMsg := fmt.Sprintf("Error Parsing time from maxRevisionDatetime  %v", err)
			return timeReq, fmt.Errorf(errMsg)
		}
		timeReq.MaxRevTime = &maxRevTime
	}
	return timeReq, nil
}

func createAppProtectLogConfEx(logConfObj *unstructured.Unstructured) (*AppProtectLogConfEx, error) {
	err := ValidateAppProtectLogConf(logConfObj)
	if err != nil {
		return &AppProtectLogConfEx{
			Obj:      logConfObj,
			IsValid:  false,
			ErrorMsg: failedValidationErrorMsg,
		}, err
	}
	return &AppProtectLogConfEx{
		Obj:     logConfObj,
		IsValid: true,
	}, nil
}

func createAppProtectUserSigEx(userSigObj *unstructured.Unstructured) (*AppProtectUserSigEx, error) {
	sTag := ""
	err := validateAppProtectUserSig(userSigObj)
	if err != nil {
		errMsg := failedValidationErrorMsg
		return &AppProtectUserSigEx{Obj: userSigObj, IsValid: false, Tag: sTag, ErrorMsg: errMsg}, fmt.Errorf(errMsg)
	}
	// Previous validation ensures there will be no errors
	tag, found, _ := unstructured.NestedString(userSigObj.Object, "spec", "tag")
	if found {
		sTag = tag
	}
	revTimeString, revTimeFound, _ := unstructured.NestedString(userSigObj.Object, "spec", "revisionDatetime")
	if revTimeFound {
		revTime, err := time.Parse(timeLayout, revTimeString)
		if err != nil {
			errMsg := invalidTimestampErrorMsg
			return &AppProtectUserSigEx{Obj: userSigObj, IsValid: false, ErrorMsg: errMsg}, fmt.Errorf(errMsg)
		}
		return &AppProtectUserSigEx{Obj: userSigObj,
			Tag:     sTag,
			RevTime: &revTime,
			IsValid: true}, nil
	}
	return &AppProtectUserSigEx{Obj: userSigObj,
		Tag:     sTag,
		IsValid: true}, nil
}

func isReqSatisfiedByUserSig(sigReq SignatureReq, sig *AppProtectUserSigEx) bool {
	if sig.Tag == "" || sig.Tag != sigReq.Tag {
		return false
	}
	if sigReq.RevTimes == nil || sig.RevTime == nil {
		return sig.Tag == sigReq.Tag
	}
	if sigReq.RevTimes.MinRevTime != nil && sigReq.RevTimes.MaxRevTime != nil {
		return sig.RevTime.Before(*sigReq.RevTimes.MaxRevTime) && sig.RevTime.After(*sigReq.RevTimes.MinRevTime)
	}
	if sigReq.RevTimes.MaxRevTime != nil && sig.RevTime.Before(*sigReq.RevTimes.MaxRevTime) {
		return true
	}
	if sigReq.RevTimes.MinRevTime != nil && sig.RevTime.After(*sigReq.RevTimes.MinRevTime) {
		return true
	}
	return false
}

func isReqSatisfiedByUserSigs(sigReq SignatureReq, sigs map[string]*AppProtectUserSigEx) bool {
	for _, sig := range sigs {
		if isReqSatisfiedByUserSig(sigReq, sig) && sig.IsValid {
			return true
		}
	}
	return false
}

func (apc *AppProtectConfiguration) verifyPolicyAgainstUserSigs(policy *AppProtectPolicyEx) bool {
	for _, sigreq := range policy.SignatureReqs {
		if !isReqSatisfiedByUserSigs(sigreq, apc.UserSigs) {
			return false
		}
	}
	return true
}

// AddOrUpdatePolicy adds or updates an App Protect Policy to App Protect Configuration
func (apc *AppProtectConfiguration) AddOrUpdatePolicy(policyObj *unstructured.Unstructured) (changes []AppProtectChange, problems []AppProtectProblem) {
	resNsName := getNsName(policyObj)
	policy, err := createAppProtectPolicyEx(policyObj)
	if err != nil {
		apc.Policies[resNsName] = policy
		return append(changes, AppProtectChange{Op: Delete, Resource: policy}),
			append(problems, AppProtectProblem{Object: policyObj, Reason: "Rejected", Message: err.Error()})
	}
	if apc.verifyPolicyAgainstUserSigs(policy) {
		apc.Policies[resNsName] = policy
		return append(changes, AppProtectChange{Op: AddOrUpdate, Resource: policy}), problems
	}
	policy.IsValid = false
	policy.ErrorMsg = missingUserSigErrorMsg
	apc.Policies[resNsName] = policy
	return append(changes, AppProtectChange{Op: Delete, Resource: policy}),
		append(problems, AppProtectProblem{Object: policyObj, Reason: "Rejected", Message: missingUserSigErrorMsg})
}

// AddOrUpdateLogConf adds or updates App Protect Log Configuration to App Protect Configuration
func (apc *AppProtectConfiguration) AddOrUpdateLogConf(logconfObj *unstructured.Unstructured) (changes []AppProtectChange, problems []AppProtectProblem) {
	resNsName := getNsName(logconfObj)
	logConf, err := createAppProtectLogConfEx(logconfObj)
	apc.LogConfs[resNsName] = logConf
	if err != nil {
		return append(changes, AppProtectChange{Op: Delete, Resource: logConf}),
			append(problems, AppProtectProblem{Object: logconfObj, Reason: "Rejected", Message: err.Error()})
	}
	return append(changes, AppProtectChange{Op: AddOrUpdate, Resource: logConf}), problems
}

// AddOrUpdateUserSig adds or updates App Protect User Defined Signature to App Protect Configuration
func (apc *AppProtectConfiguration) AddOrUpdateUserSig(userSigObj *unstructured.Unstructured) (change AppProtectUserSigChange, problems []AppProtectProblem) {
	resNsName := getNsName(userSigObj)
	userSig, err := createAppProtectUserSigEx(userSigObj)
	apc.UserSigs[resNsName] = userSig
	if err != nil {
		problems = append(problems, AppProtectProblem{Object: userSigObj, Reason: "Rejected", Message: err.Error()})
	}
	change.UserSigs = append(change.UserSigs, userSigObj)
	apc.buildUserSigChangeAndProblems(&problems, &change)

	return change, problems
}

// GetAppResource returns a pointer to an App Protect resource
func (apc *AppProtectConfiguration) GetAppResource(kind, key string) (*unstructured.Unstructured, error) {
	switch kind {
	case appProtectPolicyGVK.Kind:
		if obj, ok := apc.Policies[key]; ok {
			if obj.IsValid {
				return obj.Obj, nil
			}
			return nil, fmt.Errorf(obj.ErrorMsg)
		}
		return nil, fmt.Errorf("App Protect Policy %s not found", key)
	case appProtectLogConfGVK.Kind:
		if obj, ok := apc.LogConfs[key]; ok {
			if obj.IsValid {
				return obj.Obj, nil
			}
			return nil, fmt.Errorf(obj.ErrorMsg)
		}
		return nil, fmt.Errorf("App Protect LogConf %s not found", key)
	case appProtectUserSigGVK.Kind:
		if obj, ok := apc.UserSigs[key]; ok {
			if obj.IsValid {
				return obj.Obj, nil
			}
			return nil, fmt.Errorf(obj.ErrorMsg)
		}
		return nil, fmt.Errorf("App Protect UserSig %s not found", key)
	}
	return nil, fmt.Errorf("Unknown App Protect resource kind %s", kind)
}

// DeletePolicy deletes an App Protect Policy from App Protect Configuration
func (apc *AppProtectConfiguration) DeletePolicy(key string) (changes []AppProtectChange, problems []AppProtectProblem) {
	if _, has := apc.Policies[key]; has {
		change := AppProtectChange{Op: Delete, Resource: apc.Policies[key]}
		delete(apc.Policies, key)
		return append(changes, change), problems
	}
	return changes, problems
}

// DeleteLogConf deletes an App Protect Log Configuration from App Protect Configuration
func (apc *AppProtectConfiguration) DeleteLogConf(key string) (changes []AppProtectChange, problems []AppProtectProblem) {
	if _, has := apc.LogConfs[key]; has {
		change := AppProtectChange{Op: Delete, Resource: apc.LogConfs[key]}
		delete(apc.LogConfs, key)
		return append(changes, change), problems
	}
	return changes, problems
}

// DeleteUserSig deletes an App Protect User Defined Signature from App Protect Configuration
func (apc *AppProtectConfiguration) DeleteUserSig(key string) (change AppProtectUserSigChange, problems []AppProtectProblem) {
	if _, has := apc.UserSigs[key]; has {
		change.UserSigs = append(change.UserSigs, apc.UserSigs[key].Obj)
		delete(apc.UserSigs, key)
		apc.buildUserSigChangeAndProblems(&problems, &change)
	}
	return change, problems
}

func (apc *AppProtectConfiguration) detectDuplicateTags() (outcome [][]*AppProtectUserSigEx) {
	tmp := make(map[string][]*AppProtectUserSigEx)
	for _, sig := range apc.UserSigs {
		if val, has := tmp[sig.Tag]; has {
			if sig.ErrorMsg != failedValidationErrorMsg {
				tmp[sig.Tag] = append(val, sig)
			}
		} else {
			if sig.ErrorMsg != failedValidationErrorMsg {
				tmp[sig.Tag] = []*AppProtectUserSigEx{sig}
			}
		}
	}
	for key, vals := range tmp {
		if key != "" {
			outcome = append(outcome, vals)
		}
	}
	return outcome
}

// reconcileUserSigs verifies if tags defined in uds resorces are unique
func (apc *AppProtectConfiguration) reconcileUserSigs() (changes []AppProtectChange, problems []AppProtectProblem) {
	dupTag := apc.detectDuplicateTags()
	for _, sigs := range dupTag {
		sort.Sort(appProtectUserSigSlice(sigs))
		winner := sigs[0]
		if !winner.IsValid {
			winner.setValid()
			change := AppProtectChange{Op: AddOrUpdate, Resource: winner}
			changes = append(changes, change)
		}
		for _, sig := range sigs[1:] {
			if sig.IsValid {
				sig.setInvalid(duplicatedTagsErrorMsg)
				looserProblem := AppProtectProblem{Object: sig.Obj, Reason: "Rejected", Message: duplicatedTagsErrorMsg}
				looserChange := AppProtectChange{Op: Delete, Resource: sig}
				changes = append(changes, looserChange)
				problems = append(problems, looserProblem)
			}
		}
	}
	return changes, problems
}

func (apc *AppProtectConfiguration) verifyPolicies() (changes []AppProtectChange, problems []AppProtectProblem) {
	for _, pol := range apc.Policies {
		if !pol.IsValid && pol.ErrorMsg == missingUserSigErrorMsg {
			if apc.verifyPolicyAgainstUserSigs(pol) {
				pol.setValid()
				change := AppProtectChange{Op: AddOrUpdate, Resource: pol}
				changes = append(changes, change)
			}
		}
		if pol.IsValid {
			if !apc.verifyPolicyAgainstUserSigs(pol) {
				pol.setInvalid(missingUserSigErrorMsg)
				polProb := AppProtectProblem{Object: pol.Obj, Reason: "Rejected", Message: missingUserSigErrorMsg}
				polCh := AppProtectChange{Op: Delete, Resource: pol}
				changes = append(changes, polCh)
				problems = append(problems, polProb)
			}
		}
	}
	return changes, problems
}

func (apc *AppProtectConfiguration) getAllUserSigObjects() []*unstructured.Unstructured {
	out := []*unstructured.Unstructured{}
	for _, uds := range apc.UserSigs {
		if uds.IsValid {
			out = append(out, uds.Obj)
		}
	}
	return out
}

func (apc *AppProtectConfiguration) buildUserSigChangeAndProblems(problems *[]AppProtectProblem, udschange *AppProtectUserSigChange) {
	reconChanges, reconProblems := apc.reconcileUserSigs()
	verChanges, verProblems := apc.verifyPolicies()
	*problems = append(*problems, reconProblems...)
	*problems = append(*problems, verProblems...)
	reconChanges = append(reconChanges, verChanges...)
	for _, cha := range reconChanges {
		switch impl := cha.Resource.(type) {
		case *AppProtectPolicyEx:
			if cha.Op == Delete {
				udschange.PolicyDeletions = append(udschange.PolicyDeletions, impl.Obj)
			}
			if cha.Op == AddOrUpdate {
				udschange.PolicyAddsOrUpdates = append(udschange.PolicyAddsOrUpdates, impl.Obj)
			}
		case *AppProtectUserSigEx:
			continue
		}
	}
	udschange.UserSigs = apc.getAllUserSigObjects()
}
